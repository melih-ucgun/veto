package resources

import (
	"fmt"
	"os"
	"path/filepath"
)

type SymlinkResource struct {
	CanonicalID string
	Path        string // Linkin oluşturulacağı yer (örn: ~/.bashrc)
	Target      string // Gerçek dosya (örn: ~/dotfiles/bashrc)
}

func (s *SymlinkResource) ID() string {
	return s.CanonicalID
}

func (s *SymlinkResource) Check() (bool, error) {
	info, err := os.Lstat(s.Path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	// Dosya bir sembolik bağ mı?
	if info.Mode()&os.ModeSymlink == 0 {
		return false, nil // Normal dosya veya dizin, link değil
	}

	// Link nereye işaret ediyor?
	currentTarget, err := os.Readlink(s.Path)
	if err != nil {
		return false, err
	}

	return currentTarget == s.Target, nil
}

func (s *SymlinkResource) Diff() (string, error) {
	info, err := os.Lstat(s.Path)
	if os.IsNotExist(err) {
		return fmt.Sprintf("+ link: %s -> %s", s.Path, s.Target), nil
	}

	if info.Mode()&os.ModeSymlink == 0 {
		return fmt.Sprintf("! HATA: %s mevcut ve bir link değil!", s.Path), nil
	}

	currentTarget, _ := os.Readlink(s.Path)
	if currentTarget != s.Target {
		return fmt.Sprintf("~ link: %s (%s -> %s)", s.Path, currentTarget, s.Target), nil
	}

	return "", nil
}

func (s *SymlinkResource) Apply() error {
	// Eğer hedef dizin yoksa oluştur
	dir := filepath.Dir(s.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Mevcut bir şey varsa sil (link yanlış olabilir veya normal dosyadır)
	if _, err := os.Lstat(s.Path); err == nil {
		if err := os.Remove(s.Path); err != nil {
			return fmt.Errorf("mevcut dosya/link silinemedi: %w", err)
		}
	}

	return os.Symlink(s.Target, s.Path)
}
