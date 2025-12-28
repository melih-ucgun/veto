package resources

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

// FileResource, bir dosyanın sistemdeki durumunu temsil eder.
type FileResource struct {
	ResourceName string
	Path         string
	Content      string
}

// ID, kaynağın benzersiz kimliğini döndürür.
func (f *FileResource) ID() string {
	return fmt.Sprintf("file:%s", f.Path)
}

// Check, dosyanın şu anki halinin monarch.yaml'daki tanıma uyup uymadığını kontrol eder.
func (f *FileResource) Check() (bool, error) {
	// 1. Dosya var mı kontrol et
	info, err := os.Stat(f.Path)
	if os.IsNotExist(err) {
		// Dosya yoksa durum "uygun değil" (false)
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("dosya kontrol edilemedi: %w", err)
	}

	// 2. Eğer belirtilen yol bir dizinse hata döndür
	if info.IsDir() {
		return false, fmt.Errorf("%s bir dizin, dosya olması bekleniyordu", f.Path)
	}

	// 3. İçerik aynı mı kontrol et
	currentContent, err := os.ReadFile(f.Path)
	if err != nil {
		return false, fmt.Errorf("dosya okunamadı: %w", err)
	}

	// Mevcut içerik ile beklenen içeriği karşılaştır
	return bytes.Equal(currentContent, []byte(f.Content)), nil
}

// Apply, dosyayı istenen içeriğe getirir (Oluşturur veya günceller).
func (f *FileResource) Apply() error {
	// Dosyanın bulunduğu dizin yoksa oluştur (Örn: ~/.config/waybar/ için dizini yaratır)
	dir := filepath.Dir(f.Path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("dizin oluşturulamadı %s: %w", dir, err)
	}

	// Dosyayı yaz (0644 izinleri ile)
	err := os.WriteFile(f.Path, []byte(f.Content), 0o644)
	if err != nil {
		return fmt.Errorf("dosya yazılamadı %s: %w", f.Path, err)
	}

	return nil
}
