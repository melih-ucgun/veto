package resources

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type GitResource struct {
	CanonicalID string
	URL         string
	Path        string
}

func (g *GitResource) ID() string {
	return g.CanonicalID
}

func (g *GitResource) Check() (bool, error) {
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

	gitDir := filepath.Join(g.Path, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return false, nil
	}

	return true, nil
}

// Diff, Git reposunun durumunu raporlar.
func (g *GitResource) Diff() (string, error) {
	exists, _ := g.Check()
	if !exists {
		return fmt.Sprintf("+ git clone %s -> %s", g.URL, g.Path), nil
	}
	return fmt.Sprintf("~ git pull %s", g.Path), nil
}

func (g *GitResource) Apply() error {
	if _, err := os.Stat(g.Path); os.IsNotExist(err) {
		parentDir := filepath.Dir(g.Path)
		os.MkdirAll(parentDir, 0755)

		cmd := exec.Command("git", "clone", g.URL, g.Path)
		return cmd.Run()
	}

	cmd := exec.Command("git", "-C", g.Path, "pull")
	return cmd.Run()
}
