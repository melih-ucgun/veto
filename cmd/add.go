package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/melih-ucgun/veto/internal/config"
	"github.com/melih-ucgun/veto/internal/core"
	"github.com/melih-ucgun/veto/internal/hub"
	"github.com/melih-ucgun/veto/internal/system"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var forceType string
var asService bool

var addCmd = &cobra.Command{
	Use:   "add [resource_name/path]...",
	Short: "Add a new resource to configuration",
	Long:  `Intelligently adds resources to the active configuration profile. Detects files, packages, and services automatically.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		manager := hub.NewRecipeManager("")
		activeRecipe, _ := manager.GetActive()

		configPath := "system.yaml"
		if activeRecipe != "" {
			path, err := manager.GetRecipePath(activeRecipe)
			if err == nil {
				configPath = path
			}
		}

		// Verify config exists
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			pterm.Error.Printf("Config file '%s' not found. Run 'veto init' first.\n", configPath)
			return
		}

		pterm.Info.Printf("Adding resources to: %s\n", configPath)

		ctx := system.Detect(false)
		addedCount := 0

		for _, arg := range args {
			res := detectResource(arg, ctx)
			if res == nil {
				continue
			}

			// Check Ignore List
			ignoreMgr, _ := config.NewIgnoreManager(".vetoignore")
			if ignoreMgr != nil && ignoreMgr.IsIgnored(res.Name) {
				pterm.Warning.Printf("Resource '%s' is ignored by .vetoignore. Skipping.\n", res.Name)
				continue
			}

			if err := appendResourceToConfig(configPath, *res); err != nil {
				pterm.Error.Printf("Failed to add '%s': %v\n", arg, err)
			} else {
				pterm.Success.Printf("Added: %s (%s)\n", res.Name, res.Type)
				addedCount++
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().StringVarP(&forceType, "type", "t", "", "Force resource type (pkg, file, service)")
	addCmd.Flags().BoolVarP(&asService, "service", "s", false, "Treat as service")
}

func detectResource(input string, ctx *core.SystemContext) *config.ResourceConfig {
	// Need to fix import types mismatch if any. system.Detect returns *core.SystemContext?
	// Let's check imports. system package returns *core.SystemContext usually or its own copy?
	// Checking previous files... system.Detect returns *core.SystemContext.
	// Oh wait, I see "internal/core" imported as core in other files.
	// Here I need to import core or alias system context properly.
	// Let's assume system.Detect returns interface compatible struct or I need to import core.

	// 1. Force Type
	if forceType != "" {
		return &config.ResourceConfig{
			Type:  forceType,
			Name:  input,
			State: "present",
			Params: map[string]interface{}{
				"path": input, // Just in case it's a file
			},
		}
	}

	// 2. Service Flag
	if asService {
		return &config.ResourceConfig{
			Type:  "service",
			Name:  input,
			State: "running",
			Params: map[string]interface{}{
				"enabled": true,
			},
		}
	}

	// 3. Smart Detection

	// A. File Detection
	// Expand tilde
	expanded := input
	if strings.HasPrefix(input, "~/") {
		home, _ := os.UserHomeDir()
		expanded = filepath.Join(home, input[2:])
	}

	if info, err := os.Stat(expanded); err == nil && !info.IsDir() {
		// It is a file!
		absPath, _ := filepath.Abs(expanded)
		return &config.ResourceConfig{
			Type: "file",
			Name: filepath.Base(input),
			// Name collision risk managed by user for now
			State: "present",
			Params: map[string]interface{}{
				"path": absPath,
			},
		}
	}

	// B. Package Detection
	// Check if package manager has it installed
	// We can use discovery package helper if exposed, or just assume pkg for now
	// Ideally run "pacman -Qi input" etc.
	// For MVP, if it looks like a package (no / or .), treat as package.

	// Check simple heurustic
	if !strings.Contains(input, "/") && !strings.Contains(input, "\\") {
		// Assume package
		return &config.ResourceConfig{
			Type:  "pkg",
			Name:  input,
			State: "present",
		}
	}

	pterm.Warning.Printf("Could not detect type for '%s'. Use --type flag.\n", input)
	return nil
}

func appendResourceToConfig(path string, res config.ResourceConfig) error {
	// Read existing
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var cfg config.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return err
	}

	// Check duplicate
	for _, r := range cfg.Resources {
		if r.Type == res.Type && (r.Name == res.Name || (r.Params["path"] == res.Params["path"])) {
			return fmt.Errorf("resource already exists")
		}
	}

	// Append
	cfg.Resources = append(cfg.Resources, res)

	// Write back
	newData, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, newData, 0644)
}
