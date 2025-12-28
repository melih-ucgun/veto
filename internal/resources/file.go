package resources

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

type FileResource struct {
	CanonicalID  string
	ResourceName string
	Path         string
	Content      string
}

func (f *FileResource) ID() string {
	return f.CanonicalID
}

func (f *FileResource) Check() (bool, error) {
	info, err := os.Stat(f.Path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("dosya kontrol edilemedi: %w", err)
	}

	if info.IsDir() {
		return false, fmt.Errorf("%s bir dizin, dosya olması bekleniyordu", f.Path)
	}

	currentContent, err := os.ReadFile(f.Path)
	if err != nil {
		return false, fmt.Errorf("dosya okunamadı: %w", err)
	}

	return bytes.Equal(currentContent, []byte(f.Content)), nil
}

func (f *FileResource) Diff() (string, error) {
	current, err := os.ReadFile(f.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Sprintf("+++ %s (Yeni Dosya)\n%s", f.Path, f.Content), nil
		}
		return "", err
	}

	if string(current) != f.Content {
		return fmt.Sprintf("--- %s (Mevcut)\n+++ %s (İstenen)\n@@ İçerik değişecek @@", f.Path, f.Path), nil
	}

	return "", nil
}

func (f *FileResource) Apply() error {
	dir := filepath.Dir(f.Path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("dizin oluşturulamadı %s: %w", dir, err)
	}

	return os.WriteFile(f.Path, []byte(f.Content), 0o644)
}
