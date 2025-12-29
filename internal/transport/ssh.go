package transport

import (
	"bytes"
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
	"golang.org/x/term"
)

// RetryConfig, yeniden deneme ayarlarını tutar
const (
	MaxRetries = 3               // En fazla kaç kere denenecek
	BaseDelay  = 2 * time.Second // İlk bekleme süresi
)

type SSHTransport struct {
	client *ssh.Client
	config *ssh.ClientConfig
}

// retry, verilen fonksiyonu hata alması durumunda Exponential Backoff ile tekrar dener.
func retry(operationName string, operation func() error) error {
	var err error
	delay := BaseDelay

	for i := 0; i < MaxRetries; i++ {
		err = operation()
		if err == nil {
			return nil // Başarılı
		}

		// Son denemeyse bekleme yapma, hatayı dön
		if i == MaxRetries-1 {
			break
		}

		// Hata logu (Warning seviyesinde)
		slog.Warn("İşlem başarısız, tekrar deneniyor...",
			"islem", operationName,
			"deneme", i+1,
			"maks_deneme", MaxRetries,
			"bekleme_suresi", delay,
			"hata", err,
		)

		time.Sleep(delay)
		delay *= 2 // Süreyi ikiye katla (Exponential Backoff)
	}

	return fmt.Errorf("%s başarısız oldu (%d deneme sonrası): %w", operationName, MaxRetries, err)
}

func NewSSHTransport(host config.Host) (*SSHTransport, error) {
	sshConfig := &ssh.ClientConfig{
		User:            host.User,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	home, err := os.UserHomeDir()
	if err == nil {
		hostKeyCallback, err := knownhosts.New(filepath.Join(home, ".ssh", "known_hosts"))
		if err == nil {
			sshConfig.HostKeyCallback = hostKeyCallback
		}
	}

	authMethods := []ssh.AuthMethod{}

	// 1. SSH Agent
	if socket := os.Getenv("SSH_AUTH_SOCK"); socket != "" {
		if conn, err := net.Dial("unix", socket); err == nil {
			conn.Close() // Kaynak sızıntısını önlemek için kapatıyoruz
			// agent implementasyonu gerekirse buraya eklenebilir
		}
	}

	// 2. Private Key
	keyPath := filepath.Join(home, ".ssh", "id_rsa")
	key, err := os.ReadFile(keyPath)
	if err == nil {
		signer, err := ssh.ParsePrivateKey(key)
		if err == nil {
			authMethods = append(authMethods, ssh.PublicKeys(signer))
		}
	} else if host.Password != "" {
		// 3. Config Şifresi
		authMethods = append(authMethods, ssh.Password(host.Password))
	} else {
		// 4. İnteraktif Şifre
		fmt.Printf("%s@%s için SSH şifresi: ", host.User, host.Address)
		pass, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err == nil {
			fmt.Println()
			authMethods = append(authMethods, ssh.Password(string(pass)))
		}
	}

	sshConfig.Auth = authMethods
	addr := fmt.Sprintf("%s:%d", host.Address, host.Port)

	var client *ssh.Client

	// SSH Bağlantısını Retry mekanizması ile sarıyoruz
	connectErr := retry("SSH Bağlantısı", func() error {
		var dialErr error
		client, dialErr = ssh.Dial("tcp", addr, sshConfig)
		return dialErr
	})

	if connectErr != nil {
		return nil, connectErr
	}

	return &SSHTransport{client: client, config: sshConfig}, nil
}

func (t *SSHTransport) Close() error {
	if t.client != nil {
		return t.client.Close()
	}
	return nil
}

func (t *SSHTransport) GetRemoteSystemInfo() (string, string, error) {
	session, err := t.client.NewSession()
	if err != nil {
		return "", "", err
	}
	defer session.Close()

	var b bytes.Buffer
	session.Stdout = &b
	if err := session.Run("uname -s && uname -m"); err != nil {
		return "", "", fmt.Errorf("sistem bilgisi alınamadı: %w", err)
	}

	parts := strings.Split(strings.TrimSpace(b.String()), "\n")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("beklenmeyen sistem çıktısı: %v", parts)
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

func (t *SSHTransport) CopyFile(localPath, remotePath string) error {
	// Dosya kopyalamayı da retry mekanizması ile sarıyoruz
	return retry(fmt.Sprintf("Dosya Kopyalama (%s)", filepath.Base(localPath)), func() error {
		f, err := os.Open(localPath)
		if err != nil {
			return err // Yerel dosya hatası retry edilmemeli aslında ama connection hatası da olabilir
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

		cmd := fmt.Sprintf("scp -t %s", filepath.Dir(remotePath))
		return session.Run(cmd)
	})
}

func (t *SSHTransport) DownloadFile(remotePath, localPath string) error {
	// SFTP indirmeyi retry mekanizması ile sarıyoruz
	return retry(fmt.Sprintf("Dosya İndirme (%s)", remotePath), func() error {
		sftpClient, err := sftp.NewClient(t.client)
		if err != nil {
			return fmt.Errorf("SFTP başlatılamadı: %w", err)
		}
		defer sftpClient.Close()

		remoteFile, err := sftpClient.Open(remotePath)
		if err != nil {
			return fmt.Errorf("uzak dosya açılamadı: %w", err)
		}
		defer remoteFile.Close()

		localFile, err := os.Create(localPath)
		if err != nil {
			return fmt.Errorf("yerel dosya oluşturulamadı: %w", err)
		}
		defer localFile.Close()

		if _, err := io.Copy(localFile, remoteFile); err != nil {
			return fmt.Errorf("veri aktarım hatası: %w", err)
		}

		return localFile.Sync()
	})
}

func (t *SSHTransport) CaptureRemoteOutput(cmd string) (string, error) {
	// Komut çalıştırmayı retry etmiyoruz çünkü komut "idempotent" (tekrarlanabilir) olmayabilir.
	// Örn: "rm -rf" komutunu iki kere denersek hata alabiliriz veya veritabanına iki kere kayıt atabiliriz.
	session, err := t.client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func (t *SSHTransport) RunRemoteSecure(cmd string, becomePass string) error {
	// Sudo gerektiren komutları da riskli olduğu için retry etmiyoruz.
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

	return session.Wait()
}
