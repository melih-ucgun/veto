package transport

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/melih-ucgun/monarch/internal/config"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// RetryConfig
const (
	MaxRetries = 3
	BaseDelay  = 2 * time.Second
)

type SSHTransport struct {
	client *ssh.Client
	config *ssh.ClientConfig
}

// retry, context iptal edilirse beklemeden çıkar, yoksa backoff uygular.
func retry(ctx context.Context, operationName string, operation func() error) error {
	var err error
	delay := BaseDelay

	for i := 0; i < MaxRetries; i++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		err = operation()
		if err == nil {
			return nil
		}

		if i == MaxRetries-1 {
			break
		}

		slog.Warn("İşlem başarısız, tekrar deneniyor...",
			"islem", operationName,
			"deneme", i+1,
			"hata", err,
		)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			delay *= 2
		}
	}

	return fmt.Errorf("%s başarısız oldu: %w", operationName, err)
}

// NewSSHTransport artık güvenli Host Key doğrulaması yapıyor.
func NewSSHTransport(ctx context.Context, host config.Host) (*SSHTransport, error) {
	// 1. known_hosts dosyasını hazırla
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("home dizini bulunamadı: %w", err)
	}
	knownHostsPath := filepath.Join(home, ".ssh", "known_hosts")

	// .ssh klasörünü ve known_hosts dosyasını gerekirse oluştur
	sshDir := filepath.Dir(knownHostsPath)
	if _, err := os.Stat(sshDir); os.IsNotExist(err) {
		os.MkdirAll(sshDir, 0700)
	}
	// Dosya yoksa boş oluştur (knownhosts.New hata vermesin diye)
	if _, err := os.Stat(knownHostsPath); os.IsNotExist(err) {
		f, createErr := os.OpenFile(knownHostsPath, os.O_CREATE|os.O_WRONLY, 0600)
		if createErr != nil {
			return nil, fmt.Errorf("known_hosts dosyası oluşturulamadı: %w", createErr)
		}
		f.Close()
	}

	// 2. knownhosts Callback'ini yükle
	hkCallback, err := knownhosts.New(knownHostsPath)
	if err != nil {
		return nil, fmt.Errorf("known_hosts okunamadı: %w", err)
	}

	// 3. SSH Config Ayarları
	sshConfig := &ssh.ClientConfig{
		User:    host.User,
		Timeout: 10 * time.Second,
		// Özel HostKeyCallback: MitM koruması ve TOFU (Trust On First Use) sağlar
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			// Önce known_hosts dosyasını kontrol et
			err := hkCallback(hostname, remote, key)
			if err == nil {
				return nil // Tanınan ve güvenli sunucu
			}

			// Hata türünü incele
			var keyErr *knownhosts.KeyError
			if errors.As(err, &keyErr) {
				// Durum A: Sunucu biliniyor ama anahtar DEĞİŞMİŞ (Tehlike!)
				if len(keyErr.Want) > 0 {
					return fmt.Errorf("\n⚠️  GÜVENLİK UYARISI: '%s' sunucusunun kimliği değişmiş!\n"+
						"Bu bir Man-in-the-Middle saldırısı olabilir.\n"+
						"Bağlantı güvenlik nedeniyle reddedildi.", hostname)
				}

				// Durum B: Sunucu bilinmiyor (known_hosts içinde yok) -> Kullanıcıya sor
				return askToTrustHost(knownHostsPath, hostname, key)
			}

			// Diğer hatalar
			return err
		},
	}

	// 4. Kimlik Doğrulama Yöntemleri
	authMethods := []ssh.AuthMethod{}

	// SSH Agent
	if socket := os.Getenv("SSH_AUTH_SOCK"); socket != "" {
		if conn, err := net.Dial("unix", socket); err == nil {
			conn.Close()
			// Not: Orijinal kodda agent signers eklenmiyordu, yapıyı bozmamak için
			// burayı olduğu gibi bıraktım. Geliştirmek isterseniz agent.NewClient eklenmeli.
		}
	}

	// Private Key
	keyPath := filepath.Join(home, ".ssh", "id_rsa")
	if key, err := os.ReadFile(keyPath); err == nil {
		if signer, err := ssh.ParsePrivateKey(key); err == nil {
			authMethods = append(authMethods, ssh.PublicKeys(signer))
		}
	} else if host.Password != "" {
		authMethods = append(authMethods, ssh.Password(host.Password))
	}

	sshConfig.Auth = authMethods
	addr := fmt.Sprintf("%s:%d", host.Address, host.Port)

	var client *ssh.Client

	// Bağlantıyı kur
	connectErr := retry(ctx, "SSH Bağlantısı", func() error {
		d := net.Dialer{Timeout: sshConfig.Timeout}
		conn, err := d.DialContext(ctx, "tcp", addr)
		if err != nil {
			return err
		}

		c, chans, reqs, err := ssh.NewClientConn(conn, addr, sshConfig)
		if err != nil {
			conn.Close()
			return err
		}
		client = ssh.NewClient(c, chans, reqs)
		return nil
	})

	if connectErr != nil {
		return nil, connectErr
	}

	return &SSHTransport{client: client, config: sshConfig}, nil
}

// askToTrustHost: Bilinmeyen sunucular için kullanıcıdan onay ister ve kaydeder.
func askToTrustHost(path string, hostname string, key ssh.PublicKey) error {
	fingerprint := ssh.FingerprintSHA256(key)

	fmt.Printf("\nBu sunucunun (%s) kimliği doğrulanamadı.\n", hostname)
	fmt.Printf("Parmak İzi (Fingerprint): %s\n", fingerprint)
	fmt.Printf("Bağlantıya devam etmek istiyor musunuz? (yes/no): ")

	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.ToLower(strings.TrimSpace(response))

	if response != "yes" && response != "y" {
		return fmt.Errorf("bağlantı kullanıcı tarafından reddedildi")
	}

	// Dosyaya ekle (Append)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("known_hosts dosyasına yazılamadı: %w", err)
	}
	defer f.Close()

	// Standart known_hosts formatı: host key-type key-base64
	keyType := key.Type()
	keyBase64 := base64.StdEncoding.EncodeToString(key.Marshal())

	// Eğer hostname standart dışı bir porta sahipse, knownhosts bazen [host]:port formatını bekler
	// Basitlik için gelen hostname'i olduğu gibi yazıyoruz.
	line := fmt.Sprintf("%s %s %s\n", hostname, keyType, keyBase64)

	if _, err := f.WriteString(line); err != nil {
		return fmt.Errorf("dosyaya yazma hatası: %w", err)
	}

	fmt.Println("✅ Sunucu güvenilenler listesine eklendi.")
	return nil
}

func (t *SSHTransport) Close() error {
	if t.client != nil {
		return t.client.Close()
	}
	return nil
}

func (t *SSHTransport) GetRemoteSystemInfo(ctx context.Context) (string, string, error) {
	if ctx.Err() != nil {
		return "", "", ctx.Err()
	}

	session, err := t.client.NewSession()
	if err != nil {
		return "", "", err
	}
	defer session.Close()

	var b bytes.Buffer
	session.Stdout = &b
	if err := session.Run("uname -s && uname -m"); err != nil {
		return "", "", err
	}

	parts := strings.Split(strings.TrimSpace(b.String()), "\n")
	if len(parts) < 2 {
		return "linux", "amd64", nil
	}

	osName := strings.ToLower(parts[0])
	arch := strings.ToLower(parts[1])
	if arch == "x86_64" {
		arch = "amd64"
	} else if arch == "aarch64" {
		arch = "arm64"
	}
	return osName, arch, nil
}

func (t *SSHTransport) CopyFile(ctx context.Context, localPath, remotePath string) error {
	return retry(ctx, fmt.Sprintf("Dosya Kopyalama (%s)", filepath.Base(localPath)), func() error {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		f, err := os.Open(localPath)
		if err != nil {
			return err
		}
		defer f.Close()

		stat, err := f.Stat()
		if err != nil {
			return err
		}

		session, err := t.client.NewSession()
		if err != nil {
			return err
		}
		defer session.Close()

		go func() {
			w, _ := session.StdinPipe()
			defer w.Close()
			fmt.Fprintln(w, "C0"+fmt.Sprintf("%o", stat.Mode().Perm()), stat.Size(), filepath.Base(remotePath))
			io.Copy(w, f)
			fmt.Fprint(w, "\x00")
		}()

		done := make(chan error, 1)
		go func() {
			done <- session.Run(fmt.Sprintf("scp -t %s", filepath.Dir(remotePath)))
		}()

		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			session.Close()
			return ctx.Err()
		}
	})
}

func (t *SSHTransport) DownloadFile(ctx context.Context, remotePath, localPath string) error {
	return retry(ctx, fmt.Sprintf("Dosya İndirme (%s)", remotePath), func() error {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		sftpClient, err := sftp.NewClient(t.client)
		if err != nil {
			return err
		}
		defer sftpClient.Close()

		remoteFile, err := sftpClient.Open(remotePath)
		if err != nil {
			return err
		}
		defer remoteFile.Close()

		localFile, err := os.Create(localPath)
		if err != nil {
			return err
		}
		defer localFile.Close()

		type copyResult struct {
			n   int64
			err error
		}
		done := make(chan copyResult, 1)

		go func() {
			n, err := io.Copy(localFile, remoteFile)
			done <- copyResult{n, err}
		}()

		select {
		case res := <-done:
			if res.err == nil {
				return localFile.Sync()
			}
			return res.err
		case <-ctx.Done():
			sftpClient.Close()
			return ctx.Err()
		}
	})
}

func (t *SSHTransport) RunRemoteSecure(ctx context.Context, cmd string, becomePass string) error {
	session, err := t.client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
		return fmt.Errorf("PTY isteği başarısız: %w", err)
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		return err
	}

	finalCmd := cmd
	if t.config.User != "root" && becomePass != "" {
		finalCmd = fmt.Sprintf("sudo -S -p '' %s", cmd)
	}

	if err := session.Start(finalCmd); err != nil {
		return err
	}

	if t.config.User != "root" && becomePass != "" {
		_, _ = stdin.Write([]byte(becomePass + "\n"))
	}

	go io.Copy(os.Stdout, stdout)

	done := make(chan error, 1)
	go func() {
		done <- session.Wait()
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		slog.Warn("İşlem iptal edildi, uzak süreç sonlandırılıyor...")
		_ = session.Signal(ssh.SIGKILL)
		session.Close()
		return ctx.Err()
	}
}
