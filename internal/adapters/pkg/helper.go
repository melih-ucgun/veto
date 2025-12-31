package pkg

import (
	"os/exec"
)

// Runner interface defines methods for running commands.
// It allows mocking command execution in tests.
type Runner interface {
	Run(cmd *exec.Cmd) error
	CombinedOutput(cmd *exec.Cmd) ([]byte, error)
}

// RealRunner implements Runner using real os/exec.
type RealRunner struct{}

func (r *RealRunner) Run(cmd *exec.Cmd) error {
	return cmd.Run()
}

func (r *RealRunner) CombinedOutput(cmd *exec.Cmd) ([]byte, error) {
	return cmd.CombinedOutput()
}

// CommandRunner is the global runner instance.
// Tests can replace this with a mock.
var CommandRunner Runner = &RealRunner{}

// isInstalled, verilen komutun başarıyla çalışıp çalışmadığını kontrol eder.
// Paket yöneticileri genellikle paket varsa 0, yoksa hata kodu döner.
func isInstalled(checkCmd string, args ...string) bool {
	cmd := exec.Command(checkCmd, args...)
	if err := CommandRunner.Run(cmd); err != nil {
		return false
	}
	return true
}

// runCommand, bir komutu çalıştırır ve çıktısını/hatasını döner.
func runCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := CommandRunner.CombinedOutput(cmd)
	return string(out), err
}
