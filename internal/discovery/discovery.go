package discovery

import (
	"fmt"

	"github.com/melih-ucgun/monarch/internal/config"
	"github.com/melih-ucgun/monarch/internal/core"
)

// DiscoverSystem scans the system and returns a generated configuration.
func DiscoverSystem(ctx *core.SystemContext) (*config.Config, error) {
	cfg := &config.Config{
		Resources: []config.ResourceConfig{},
	}

	// 1. Discover Packages
	pkgs, err := discoverPackages(ctx)
	if err != nil {
		return nil, fmt.Errorf("package discovery failed: %w", err)
	}

	for _, pkgName := range pkgs {
		cfg.Resources = append(cfg.Resources, config.ResourceConfig{
			Type:  "pkg",
			Name:  pkgName,
			State: "present",
		})
	}

	// 2. Discover Services
	services, err := discoverServices(ctx.InitSystem)
	if err != nil {
		// Log error but continue? Or fail?
		// Let's log and continue for partial result
		fmt.Printf("Warning: Service discovery failed: %v\n", err)
	}

	for _, svcName := range services {
		cfg.Resources = append(cfg.Resources, config.ResourceConfig{
			Type:  "service",
			Name:  svcName,
			State: "running", // or enabled
			Params: map[string]interface{}{
				"enabled": true,
			},
		})
	}

	// Add comments/metadata if possible (YAML marshaler might not support comments easily)

	return cfg, nil
}
