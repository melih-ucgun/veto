package core

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// BackupManager, dosya yedekleme işlemlerini yönetir.
type BackupManager struct {
	BackupDir string
}

// Global backup instance (basitlik için)
var GlobalBackup *BackupManager

// InitBackupManager, yedekleme dizinini hazırlar.
func InitBackupManager() error {
	// .monarch/backups/YYYYMMDD-HHMMSS formatında klasör oluştur
	timestamp := time.Now().Format("20060102-150405")
	backupDir := filepath.Join(".monarch", "backups", timestamp)

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup dir: %w", err)
	}

	GlobalBackup = &BackupManager{BackupDir: backupDir}
	return nil
}

// BackupFile, verilen dosyayı yedekler ve yedek yolunu döner.
// Eğer dosya yoksa (yeni oluşturulacaksa), boş string döner (hata değil).
func (bm *BackupManager) BackupFile(path string) (string, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return "", nil // Dosya yok, yedeklemeye gerek yok
	}
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		// Klasör yedekleme şu an kapsam dışı (veya recursive yapılabilir)
		return "", nil
	}

	// Yedek dosya yolu: backups/<timestamp>/hash_filename
	// Hash kullanarak path çakışmalarını önlüyoruz (örn: /etc/hosts vs /tmp/hosts)
	hasher := sha256.New()
	hasher.Write([]byte(path))
	pathHash := hex.EncodeToString(hasher.Sum(nil))[:8]

	flatFilename := fmt.Sprintf("%s_%s", pathHash, filepath.Base(path))
	backupPath := filepath.Join(bm.BackupDir, flatFilename)

	// Dosyayı kopyala
	srcFile, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer srcFile.Close()

	destFile, err := os.Create(backupPath)
	if err != nil {
		return "", err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, srcFile); err != nil {
		return "", err
	}

	return backupPath, nil
}
