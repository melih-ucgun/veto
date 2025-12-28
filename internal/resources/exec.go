package resources

import (
	"fmt"
	"os/exec"
)

// ExecResource, sistemde doğrudan komut çalıştırmak için kullanılır.
type ExecResource struct {
	Name    string
	Command string
}

func (e *ExecResource) ID() string {
	return fmt.Sprintf("exec:%s", e.Name)
}

// Check metodu exec için her zaman false dönebilir (her seferinde çalışması isteniyorsa)
// veya komutun daha önce başarılı olup olmadığını kontrol eden bir mantık eklenebilir.
func (e *ExecResource) Check() (bool, error) {
	// Şimdilik "idempotent" komutlar varsayılarak her zaman çalıştırılması için false dönüyoruz.
	return false, nil
}

func (e *ExecResource) Apply() error {
	// bash -c kullanarak kompleks komutları (pipe vb.) destekliyoruz.
	cmd := exec.Command("bash", "-c", e.Command)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("komut çalıştırma hatası [%s]: %w", e.Name, err)
	}
	return nil
}
