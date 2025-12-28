package transport

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/melih-ucgun/monarch/internal/config"
	"golang.org/x/crypto/ssh"
)

type SSHTransport struct {
	client *ssh.Client
	host   config.Host
}

func NewSSHTransport(h config.Host) (*SSHTransport, error) {
	var authMethods []ssh.AuthMethod
	if h.Password != "" {
		authMethods = append(authMethods, ssh.Password(h.Password))
	}

	clientConfig := &ssh.ClientConfig{
		User:            h.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}

	port := h.Port
	if port == 0 {
		port = 22
	}

	addr := fmt.Sprintf("%s:%d", h.Address, port)
	client, err := ssh.Dial("tcp", addr, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("SSH bağlantısı kurulamadı: %w", err)
	}

	return &SSHTransport{client: client, host: h}, nil
}

func (t *SSHTransport) GetRemoteSystemInfo() (string, string, error) {
	osOut, err := t.CaptureRemoteOutput("uname -s")
	if err != nil {
		return "", "", err
	}
	remoteOS := strings.ToLower(strings.TrimSpace(osOut))

	archOut, err := t.CaptureRemoteOutput("uname -m")
	if err != nil {
		return "", "", err
	}
	rawArch := strings.TrimSpace(archOut)

	remoteArch := "amd64"
	switch rawArch {
	case "x86_64":
		remoteArch = "amd64"
	case "aarch64", "arm64":
		remoteArch = "arm64"
	}

	return remoteOS, remoteArch, nil
}

func (t *SSHTransport) CopyFile(srcPath, destPath string) error {
	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer f.Close()

	stat, _ := f.Stat()
	session, err := t.client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	go func() {
		w, _ := session.StdinPipe()
		defer w.Close()
		fmt.Fprintf(w, "C%04o %d %s\n", stat.Mode().Perm(), stat.Size(), "file")
		io.Copy(w, f)
		fmt.Fprint(w, "\x00")
	}()

	return session.Run(fmt.Sprintf("scp -t %s", destPath))
}

func (t *SSHTransport) CaptureRemoteOutput(cmd string) (string, error) {
	session, err := t.client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	var stdout bytes.Buffer
	session.Stdout = &stdout
	if err := session.Run(cmd); err != nil {
		return "", err
	}
	return stdout.String(), nil
}

func (t *SSHTransport) RunRemoteSecure(cmd string, sudoPassword string) error {
	session, err := t.client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	if sudoPassword != "" {
		in, err := session.StdinPipe()
		if err != nil {
			return err
		}
		if err := session.Start(fmt.Sprintf("sudo -S -p '' %s", cmd)); err != nil {
			return err
		}
		fmt.Fprintln(in, sudoPassword)
		return session.Wait()
	}

	return session.Run(cmd)
}

func (t *SSHTransport) Close() error {
	if t.client != nil {
		return t.client.Close()
	}
	return nil
}
