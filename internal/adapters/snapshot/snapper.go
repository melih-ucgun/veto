package snapshot

import (
	"fmt"
	"strings"

	"github.com/melih-ucgun/veto/internal/core"
	"github.com/pterm/pterm"
)

// Snapper manages integration with the snapper tool
type Snapper struct {
	configName string
}

func NewSnapper() *Snapper {
	return &Snapper{
		configName: "root",
	}
}

func (s *Snapper) Name() string {
	return "Snapper"
}

func (s *Snapper) IsAvailable(ctx *core.SystemContext) bool {
	_, err := ctx.Transport.Execute(ctx.Context, "which snapper")
	return err == nil
}

func (s *Snapper) CreateSnapshot(ctx *core.SystemContext, description string) error {
	fullCmd := fmt.Sprintf("snapper -c %s create -d \"%s\"", s.configName, description)
	_, err := ctx.Transport.Execute(ctx.Context, fullCmd)
	return err
}

func (s *Snapper) CreatePreSnapshot(ctx *core.SystemContext, description string) (string, error) {
	fullCmd := fmt.Sprintf("snapper -c %s create -t pre -p -d \"%s\"", s.configName, description)
	out, err := ctx.Transport.Execute(ctx.Context, fullCmd)
	if err != nil {
		return "", fmt.Errorf("snapper pre failed: %w", err)
	}
	return strings.TrimSpace(out), nil
}

func (s *Snapper) CreatePostSnapshot(ctx *core.SystemContext, id string, description string) error {
	if id == "" {
		return fmt.Errorf("invalid pre-snapshot id")
	}
	fullCmd := fmt.Sprintf("snapper -c %s create -t post --pre-number %s -d \"%s\"", s.configName, id, description)
	if _, err := ctx.Transport.Execute(ctx.Context, fullCmd); err != nil {
		return fmt.Errorf("snapper post failed: %w", err)
	}
	pterm.Success.Printf("Snapper pair created (Pre: %s)\n", id)
	return nil
}
