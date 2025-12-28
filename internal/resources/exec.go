package resources

import (
	"fmt"
	"os/exec"
)

type ExecResource struct {
	CanonicalID string
	Name        string
	Command     string
}

func (e *ExecResource) ID() string {
	return e.CanonicalID
}

func (e *ExecResource) Check() (bool, error) {
	// Exec kaynakları doğası gereği her zaman 'uygulanması gereken' durumda kabul edilir.
	return false, nil
}

// Diff, çalıştırılacak komutu gösterir.
func (e *ExecResource) Diff() (string, error) {
	return fmt.Sprintf("! exec: %s (Komut: %s)", e.Name, e.Command), nil
}

func (e *ExecResource) Apply() error {
	cmd := exec.Command("bash", "-c", e.Command)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("komut çalıştırma hatası [%s]: %w", e.Name, err)
	}
	return nil
}
