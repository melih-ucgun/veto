package transport

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/melih-ucgun/monarch/internal/config"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// --- MOCK SSH SERVER HELPERS ---

// generateSigner, test sunucusu için anlık bir RSA anahtarı üretir.
func generateSigner() (ssh.Signer, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	return ssh.ParsePrivateKey(keyPEM)
}

// startMockSSHServer, belirtilen handler ile basit bir SSH sunucusu başlatır.
// Geriye sunucunun dinlediği adresi (host:port) ve kapatma fonksiyonunu döner.
func startMockSSHServer(t *testing.T, handler func(ssh.Channel, <-chan *ssh.Request)) (string, func()) {
	signer, err := generateSigner()
	if err != nil {
		t.Fatalf("SSH anahtarı üretilemedi: %v", err)
	}

	// 'config' paket ismiyle çakışmaması için değişken adını 'serverConfig' yaptık
	serverConfig := &ssh.ServerConfig{
		NoClientAuth: true, // Test için şifre sorma (Client ne gönderirse kabul et)
	}
	serverConfig.AddHostKey(signer)

	listener, err := net.Listen("tcp", "127.0.0.1:0") // Rastgele port
	if err != nil {
		t.Fatalf("Dinleyici başlatılamadı: %v", err)
	}

	go func() {
		for {
			nConn, err := listener.Accept()
			if err != nil {
				return // Listener kapandı
			}

			go func(conn net.Conn) {
				_, chans, reqs, err := ssh.NewServerConn(conn, serverConfig)
				if err != nil {
					return
				}
				// Global requestleri yut
				go ssh.DiscardRequests(reqs)

				// Kanalları işle
				for newChannel := range chans {
					if newChannel.ChannelType() != "session" {
						newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
						continue
					}
					channel, requests, err := newChannel.Accept()
					if err != nil {
						continue
					}

					// Custom handler'ı çağır
					go handler(channel, requests)
				}
			}(nConn)
		}
	}()

	return listener.Addr().String(), func() { listener.Close() }
}

// --- TESTLER ---

func TestSSHTransport_RunRemoteSecure(t *testing.T) {
	// Mock Sunucu Mantığı
	addr, close := startMockSSHServer(t, func(channel ssh.Channel, reqs <-chan *ssh.Request) {
		defer channel.Close()
		for req := range reqs {
			switch req.Type {
			case "exec":
				// Komut "echo hello" ise çıktı ver
				if strings.Contains(string(req.Payload), "echo hello") {
					channel.Write([]byte("hello\n"))
				}

				req.Reply(true, nil)
				// Exit status 0 gönder
				channel.SendRequest("exit-status", false, []byte{0, 0, 0, 0})
				return
			case "pty-req":
				req.Reply(true, nil)
			default:
				req.Reply(false, nil)
			}
		}
	})
	defer close()

	// Transport Ayarları
	t.Setenv("HOME", t.TempDir()) // Private key okumasını engelle

	host, port, _ := net.SplitHostPort(addr)
	p, _ := net.LookupPort("tcp", port)

	cfgHost := config.Host{
		User:     "testuser",
		Address:  host,
		Port:     p,
		Password: "dummy-password", // ÖNEMLİ: Şifre vermezsek terminalden sormaya çalışır ve test kilitlenir!
	}

	ctx := context.Background()
	tr, err := NewSSHTransport(ctx, cfgHost)
	if err != nil {
		t.Fatalf("Transport oluşturulamadı: %v", err)
	}
	defer tr.Close()

	// Test: Komut Çalıştırma
	err = tr.RunRemoteSecure(ctx, "echo hello", "")
	if err != nil {
		t.Errorf("RunRemoteSecure hata verdi: %v", err)
	}
}

func TestSSHTransport_DownloadFile(t *testing.T) {
	// SFTP destekli Mock Sunucu
	addr, close := startMockSSHServer(t, func(channel ssh.Channel, reqs <-chan *ssh.Request) {
		serverOptions := []sftp.ServerOption{
			sftp.WithDebug(io.Discard),
		}

		for req := range reqs {
			if req.Type == "subsystem" && string(req.Payload[4:]) == "sftp" {
				req.Reply(true, nil)
				server, _ := sftp.NewServer(channel, serverOptions...)
				if err := server.Serve(); err == io.EOF {
					server.Close()
				}
				return
			}
			req.Reply(false, nil)
		}
	})
	defer close()

	t.Setenv("HOME", t.TempDir())
	host, port, _ := net.SplitHostPort(addr)
	p, _ := net.LookupPort("tcp", port)

	// Password ekledik
	ctx := context.Background()
	tr, err := NewSSHTransport(ctx, config.Host{User: "test", Address: host, Port: p, Password: "dummy"})
	if err != nil {
		t.Fatalf("Transport hatası: %v", err)
	}
	defer tr.Close()

	// 1. İndirilecek gerçek bir dosya oluştur
	tempDir := t.TempDir()
	srcContent := "bu bir test dosyasidir"
	srcPath := filepath.Join(tempDir, "remote_file.txt")
	os.WriteFile(srcPath, []byte(srcContent), 0644)

	destPath := filepath.Join(tempDir, "local_downloaded.txt")

	// 2. DownloadFile çağır
	err = tr.DownloadFile(ctx, srcPath, destPath)
	if err != nil {
		t.Fatalf("Dosya indirilemedi: %v", err)
	}

	// 3. İçeriği doğrula
	got, _ := os.ReadFile(destPath)
	if string(got) != srcContent {
		t.Errorf("İçerik hatalı. Beklenen: %s, Gelen: %s", srcContent, string(got))
	}
}

func TestSSHTransport_CopyFile_SCP(t *testing.T) {
	receivedContent := new(strings.Builder)

	addr, close := startMockSSHServer(t, func(channel ssh.Channel, reqs <-chan *ssh.Request) {
		defer channel.Close()
		for req := range reqs {
			if req.Type == "exec" {
				cmd := string(req.Payload[4:])
				if strings.Contains(cmd, "scp -t") {
					req.Reply(true, nil)
					io.Copy(receivedContent, channel)
					channel.SendRequest("exit-status", false, []byte{0, 0, 0, 0})
					return
				}
			}
			req.Reply(false, nil)
		}
	})
	defer close()

	t.Setenv("HOME", t.TempDir())
	host, port, _ := net.SplitHostPort(addr)
	p, _ := net.LookupPort("tcp", port)

	// Password ekledik
	tr, _ := NewSSHTransport(context.Background(), config.Host{User: "test", Address: host, Port: p, Password: "dummy"})

	// Tr nil değilse kapat
	if tr != nil {
		defer tr.Close()
	}

	srcFile := filepath.Join(t.TempDir(), "upload_test.txt")
	content := "SCP Data Transfer"
	os.WriteFile(srcFile, []byte(content), 0644)

	err := tr.CopyFile(context.Background(), srcFile, "/tmp/remote_dest")
	if err != nil {
		t.Errorf("CopyFile hatası: %v", err)
	}

	if !strings.Contains(receivedContent.String(), content) {
		t.Errorf("Sunucuya veri ulaşmadı. Alınan ham veri: %s", receivedContent.String())
	}
}
