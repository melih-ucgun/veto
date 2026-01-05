package identity

import (
	"github.com/melih-ucgun/veto/internal/core"
)

// GetIdentityProvider returns the appropriate provider for the current OS.
func GetIdentityProvider(ctx *core.SystemContext) IdentityProvider {
	// Future: switch ctx.OS { case "darwin": ... }
	return &LinuxIdentityProvider{}
}
