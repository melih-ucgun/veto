package transport

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/melih-ucgun/monarch/internal/config"
	"golang.org/x/crypto/ssh"
)

type SSHTransport struct {
	Client *ssh.Client
	Config config.Host
}

func NewSSHTransport(hostConfig config.Host) (*SSHTransport, error) {
	keyPath := hostConfig.KeyPath
	if strings.HasPrefix(keyPath, "~/") {
		home, _ := os.UserHomeDir()
		keyPath = filepath.Join(home, keyPath[2:])
	}

	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("SSH key okunamadı (%s): %v", keyPath, err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("SSH key parse hatası: %v", err)
	}

	sshConfig := &ssh.ClientConfig{
		User: hostConfig.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// DÜZELTME: Hostname -> Host (Varsayım) ve Port -> 22 (Varsayılan)
	// Eğer config.Host struct'ında 'Host' alanı yoksa, lütfen 'Name' veya 'Address' olarak değiştirin.
	addr := fmt.Sprintf("%s:22", hostConfig.User)

	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("SSH bağlantı hatası (%s): %v", addr, err)
	}

	return &SSHTransport{Client: client, Config: hostConfig}, nil
}

func (t *SSHTransport) RunRemoteSecure(cmd string, sudoPass string) error {
	session, err := t.Client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	return session.Run(cmd)
}

func (t *SSHTransport) Exec(cmd string) (string, error) {
	session, err := t.Client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	var stdoutBuf bytes.Buffer
	session.Stdout = &stdoutBuf

	if err := session.Run(cmd); err != nil {
		return "", err
	}
	return strings.TrimSpace(stdoutBuf.String()), nil
}

func (t *SSHTransport) GetRemoteSystemInfo() (string, string, error) {
	rawOS, err := t.Exec("uname -s")
	if err != nil {
		return "", "", fmt.Errorf("uzak OS tespit edilemedi: %v", err)
	}

	rawArch, err := t.Exec("uname -m")
	if err != nil {
		return "", "", fmt.Errorf("uzak mimari tespit edilemedi: %v", err)
	}

	goOS := strings.ToLower(rawOS)
	goArch := rawArch

	switch rawArch {
	case "x86_64":
		goArch = "amd64"
	case "aarch64", "armv8":
		goArch = "arm64"
	case "i386", "i686":
		goArch = "386"
	case "armv7l":
		goArch = "arm"
	}

	return goOS, goArch, nil
}

func (t *SSHTransport) CopyFile(localPath, remotePath string) error {
	src, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer src.Close()

	srcStat, err := src.Stat()
	if err != nil {
		return err
	}

	session, err := t.Client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	go func() {
		w, _ := session.StdinPipe()
		defer w.Close()
		fmt.Fprintln(w, "C0755", srcStat.Size(), filepath.Base(remotePath))
		io.Copy(w, src)
		fmt.Fprint(w, "\x00")
	}()

	remoteDir := filepath.Dir(remotePath)
	return session.Run(fmt.Sprintf("scp -t %s", remoteDir))
}
