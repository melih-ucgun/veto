package pkg

import (
	"fmt"
	"strings"

	"github.com/melih-ucgun/veto/internal/core"
)

// PacmanAdapter, Arch Linux pacman paket yöneticisi için adaptör.
type PacmanAdapter struct {
	core.BaseResource        // Ortak alanlar (Name, Type) buradan gelir
	State             string // "present", "absent"
	ActionPerformed   string // "installed", "removed", ""
}

// NewPacmanAdapter yeni bir örnek oluşturur.
func NewPacmanAdapter(name string, state string) *PacmanAdapter {
	if state == "" {
		state = "present"
	}
	return &PacmanAdapter{
		BaseResource: core.BaseResource{Name: name, Type: "package"},
		State:        state,
	}
}

func (r *PacmanAdapter) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("package name is required for pacman")
	}
	return nil
}

func (r *PacmanAdapter) Check(ctx *core.SystemContext) (bool, error) {
	// Pacman ile paket kontrolü: pacman -Qi <paket>
	installed := isInstalled("pacman", "-Qi", r.Name)

	if r.State == "absent" {
		// Eğer silinmesi isteniyorsa ama yüklüyse -> Değişiklik lazım (true)
		return installed, nil
	}

	// Eğer yüklenmesi isteniyorsa (present) ve yüklü değilse -> Değişiklik lazım (true)
	return !installed, nil
}

func (r *PacmanAdapter) Apply(ctx *core.SystemContext) (core.Result, error) {
	// Önce durum kontrolü yap (Dry-run desteği için önemli)
	needsAction, err := r.Check(ctx)
	if err != nil {
		return core.Failure(err, "Failed to check package status"), err
	}

	if !needsAction {
		return core.SuccessNoChange(fmt.Sprintf("Package %s is already in desired state (%s)", r.Name, r.State)), nil
	}

	if ctx.DryRun {
		return core.SuccessChange(fmt.Sprintf("[DryRun] Would %s package %s", r.State, r.Name)), nil
	}

	// İşlemi Gerçekleştir
	var cmd string
	var args []string

	if r.State == "absent" {
		// Kaldır: pacman -Rns --noconfirm
		cmd = "pacman"
		args = []string{"-Rns", "--noconfirm", r.Name}
		r.ActionPerformed = "removed"
	} else {
		// Kur: pacman -S --noconfirm --needed
		cmd = "pacman"
		args = []string{"-S", "--noconfirm", "--needed", r.Name}
		r.ActionPerformed = "installed"
	}

	output, err := runCommand(cmd, args...)
	if err != nil {
		r.ActionPerformed = ""
		return core.Failure(err, fmt.Sprintf("Failed to %s package %s: %s", r.State, r.Name, output)), err
	}

	return core.SuccessChange(fmt.Sprintf("Successfully %s package %s", r.State, r.Name)), nil
}

func (r *PacmanAdapter) Revert(ctx *core.SystemContext) error {
	if r.ActionPerformed == "installed" {
		_, err := runCommand("pacman", "-Rns", "--noconfirm", r.Name)
		return err
	}
	return nil
}

// ListInstalled returns a list of explicitly installed packages.
func (r *PacmanAdapter) ListInstalled(ctx *core.SystemContext) ([]string, error) {
	// pacman -Qqe: Query, Quiet (name only), Explicitly installed
	output, err := runCommand("pacman", "-Qqe")
	if err != nil {
		return nil, fmt.Errorf("failed to list installed packages: %w", err)
	}

	// runCommand returns string.
	// We need strings package.
	// But let's verify if runCommand output is clean. Yes standard stdout.
	return splitLines(output), nil
}

func splitLines(s string) []string {
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			lines = append(lines, trimmed)
		}
	}
	return lines
}
