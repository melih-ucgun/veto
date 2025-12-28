package transport

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/melih-ucgun/monarch/internal/config"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

// SSHTransport, uzak sunucu ile güvenli iletişim kuran ana yapıdır.
type SSHTransport struct {
	Host   config.Host
	Client *ssh.Client
}

// NewSSHTransport, bilinen host doğrulaması ve gelişmiş kimlik doğrulama ile bağlantı kurar.
func NewSSHTransport(h config.Host) (*SSHTransport, error) {
	var authMethods []ssh.AuthMethod

	// 1. SSH Agent Desteği
	if socket := os.Getenv("SSH_AUTH_SOCK"); socket != "" {
		conn, err := net.Dial("unix", socket)
		if err == nil {
			authMethods = append(authMethods, ssh.PublicKeysCallback(agent.NewClient(conn).Signers))
		}
	}

	// 2. Private Key Desteği
	if h.KeyPath != "" {
		expandedPath := expandHome(h.KeyPath)
		key, err := os.ReadFile(expandedPath)
		if err != nil {
			return nil, fmt.Errorf("SSH anahtarı okunamadı (%s): %w", expandedPath, err)
		}

		var signer ssh.Signer
		if h.Passphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(h.Passphrase))
		} else {
			signer, err = ssh.ParsePrivateKey(key)
		}

		if err != nil {
			return nil, fmt.Errorf("SSH anahtarı ayrıştırılamadı: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	// 3. Şifre Desteği
	if h.Password != "" {
		authMethods = append(authMethods, ssh.Password(h.Password))
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("hiçbir kimlik doğrulama yöntemi (Agent, Key veya Password) bulunamadı")
	}

	// 4. Host Key Doğrulaması (Hardening)
	hostKeyCallback, err := getHostKeyCallback()
	if err != nil {
		// Eğer known_hosts bulunamazsa güvenlik için hata döndürüyoruz.
		// Geliştirme aşamasında esneklik istenirse ssh.InsecureIgnoreHostKey()'e fallback yapılabilir.
		return nil, fmt.Errorf("host key doğrulaması hazırlanamadı: %w", err)
	}

	clientConfig := &ssh.ClientConfig{
		User:            h.User,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         15 * time.Second, // Zaman aşımı süresi eklendi
	}

	addr := h.Address
	if !strings.Contains(addr, ":") {
		addr = addr + ":22"
	}

	// Bağlantı kurarken daha spesifik hata kontrolleri
	client, err := ssh.Dial("tcp", addr, clientConfig)
	if err != nil {
		if strings.Contains(err.Error(), "unable to authenticate") {
			return nil, fmt.Errorf("yetkilendirme hatası: Kullanıcı adı veya anahtar geçersiz (%s)", addr)
		}
		if strings.Contains(err.Error(), "host key mismatch") {
			return nil, fmt.Errorf("GÜVENLİK UYARISI: Host anahtarı eşleşmiyor! (Man-in-the-middle saldırısı olabilir): %v", err)
		}
		return nil, fmt.Errorf("SSH bağlantısı kurulamadı (%s): %v", addr, err)
	}

	return &SSHTransport{Host: h, Client: client}, nil
}

// getHostKeyCallback, ~/.ssh/known_hosts dosyasını okur ve doğrular.
func getHostKeyCallback() (ssh.HostKeyCallback, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	knownHostsPath := filepath.Join(home, ".ssh", "known_hosts")

	// Dosya yoksa oluştur (Boş olması güvenliği bozmaz, sadece ilk bağlantıda hata verir)
	if _, err := os.Stat(knownHostsPath); os.IsNotExist(err) {
		os.MkdirAll(filepath.Dir(knownHostsPath), 0700)
		os.WriteFile(knownHostsPath, []byte(""), 0600)
	}

	callback, err := knownhosts.New(knownHostsPath)
	if err != nil {
		return nil, fmt.Errorf("known_hosts dosyası işlenemedi: %w", err)
	}

	return callback, nil
}

func (s *SSHTransport) RunRemote(command string) error {
	session, err := s.Client.NewSession()
	if err != nil {
		return fmt.Errorf("uzak oturum açılamadı: %w", err)
	}
	defer session.Close()

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	return session.Run(command)
}

func (s *SSHTransport) CopyFile(localPath, remotePath string) error {
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("yerel dosya açılamadı: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("dosya bilgileri alınamadı: %w", err)
	}

	session, err := s.Client.NewSession()
	if err != nil {
		return fmt.Errorf("SCP oturumu açılamadı: %w", err)
	}
	defer session.Close()

	go func() {
		w, err := session.StdinPipe()
		if err != nil {
			return
		}
		defer w.Close()

		fmt.Fprintf(w, "C%04o %d %s\n", 0755, stat.Size(), filepath.Base(localPath))
		io.Copy(w, file)
		fmt.Fprint(w, "\x00")
	}()

	remoteDir := filepath.Dir(remotePath)
	if err := session.Run(fmt.Sprintf("/usr/bin/scp -t %s", remoteDir)); err != nil {
		return fmt.Errorf("dosya kopyalama başarısız: %w", err)
	}

	return nil
}

func expandHome(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[1:])
		}
	}
	return path
}
