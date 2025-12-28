package resources

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// GitResource, bir Git reposunun belirli bir dizinde bulunmasını sağlar.
type GitResource struct {
	URL  string
	Path string
}

func (g *GitResource) ID() string {
	return fmt.Sprintf("git:%s", g.Path)
}

func (g *GitResource) Check() (bool, error) {
	// 1. Dizin var mı?
	info, err := os.Stat(g.Path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if !info.IsDir() {
		return false, fmt.Errorf("%s bir dizin değil", g.Path)
	}

	// 2. .git klasörü var mı?
	gitDir := filepath.Join(g.Path, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return false, nil
	}

	// 3. Remote URL doğru mu? (Basit kontrol)
	cmd := exec.Command("git", "-C", g.Path, "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return false, nil // origin yoksa veya git komutu başarısızsa
	}

	// TODO: URL karşılaştırması yapılabilir (output temizlenerek)
	_ = output

	return true, nil
}

func (g *GitResource) Apply() error {
	// Dizin yoksa klonla
	if _, err := os.Stat(g.Path); os.IsNotExist(err) {
		parentDir := filepath.Dir(g.Path)
		os.MkdirAll(parentDir, 0755)

		fmt.Printf("📥 Repo klonlanıyor: %s -> %s\n", g.URL, g.Path)
		cmd := exec.Command("git", "clone", g.URL, g.Path)
		return cmd.Run()
	}

	// Dizin varsa pull yap (State "latest" ise bu geliştirilebilir)
	fmt.Printf("🔄 Repo güncelleniyor: %s\n", g.Path)
	cmd := exec.Command("git", "-C", g.Path, "pull")
	return cmd.Run()
}
