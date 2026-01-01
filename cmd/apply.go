package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/melih-ucgun/veto/internal/adapters/snapshot"
	"github.com/melih-ucgun/veto/internal/config"
	"github.com/melih-ucgun/veto/internal/core"
	"github.com/melih-ucgun/veto/internal/hub"
	"github.com/melih-ucgun/veto/internal/resource"
	"github.com/melih-ucgun/veto/internal/state" // New import
	"github.com/melih-ucgun/veto/internal/system"
)

var dryRun bool
var noSnapshot bool
var pruneMode bool

var applyCmd = &cobra.Command{
	Use:   "apply [config_file]",
	Short: "Apply the configuration to the system",
	Long: `Reads the configuration file and ensures system state matches desired state.
Updates .veto/state.json with the results.`,
	Run: func(cmd *cobra.Command, args []string) {
		var configFile string
		if len(args) > 0 {
			configFile = args[0]
		} else {
			// Check active recipe
			mgr := hub.NewRecipeManager("")
			recipePath, err := mgr.GetRecipePath("")
			if err == nil && recipePath != "" {
				// Verify file exists
				if _, err := os.Stat(recipePath); err == nil {
					configFile = recipePath
					pterm.Info.Printf("Using active recipe: %s\n", recipePath)
				}
			}

			// Fallback
			if configFile == "" {
				configFile = "system.yaml"
				pterm.Warning.Println("No active recipe found. Defaulting to system.yaml")
			}
		}

		if err := runApply(configFile, dryRun, noSnapshot, pruneMode); err != nil {
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(applyCmd)
	applyCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Simulate changes without applying them")
	applyCmd.Flags().BoolVar(&noSnapshot, "no-snapshot", false, "Disable automatic BTRFS snapshots")
	applyCmd.Flags().BoolVar(&pruneMode, "prune", false, "Remove unmanaged package resources (DESTRUCTIVE)")
}

func runApply(configFile string, isDryRun bool, skipSnapshot bool, isPrune bool) error {
	// Header
	pterm.DefaultHeader.WithFullWidth().WithBackgroundStyle(pterm.NewStyle(pterm.BgLightBlue)).
		WithTextStyle(pterm.NewStyle(pterm.FgBlack, pterm.Bold)).
		Println("Veto Config Manager")

	if isDryRun {
		pterm.ThemeDefault.SecondaryStyle.Println("Running in DRY-RUN mode")
	}

	// 1. Detect System
	ctx := system.Detect(isDryRun)

	// 1.5 Load System Profile (if exists)
	if data, err := os.ReadFile(".veto/system.yaml"); err == nil {
		pterm.Info.Println("Loading system profile from .veto/system.yaml")
		// Override detected context with saved profile
		if err := yaml.Unmarshal(data, ctx); err != nil {
			pterm.Warning.Printf("Failed to parse system profile: %v\n", err)
		}
	}

	// Snapshot Manager Setup
	var snapMgr *snapshot.Manager
	var preSnapID string

	// Check if we should attempt a snapshot
	if !isDryRun && !skipSnapshot {
		// NewManager akıllıca seçim yapar (Snapper > Timeshift)
		snapMgr = snapshot.NewManager(ctx.FS.RootFSType)

		if snapMgr != nil && snapMgr.IsAvailable() {
			pterm.Info.Printf("Snapshot System: %s detected\n", snapMgr.ProviderName())

			id, err := snapMgr.CreatePreSnapshot(fmt.Sprintf("Pre-Veto Apply: %s", configFile))
			if err != nil {
				pterm.Warning.Printf("Snapshot failed: %v (continuing anyway)\n", err)
			} else {
				preSnapID = id
				// Timeshift 'done' döndürür, Snapper ID döndürür.
				if id != "done" {
					pterm.Success.Printf("Pre-snapshot created: #%s\n", id)
				}
			}
		}
	}

	// System Info Box
	sysInfo := [][]string{
		{"OS", fmt.Sprintf("%s (%s)", ctx.Distro, ctx.Version)},
		{"Kernel", ctx.OS},
		{"Host", ctx.Hostname},
		{"User", fmt.Sprintf("%s (uid=%s)", ctx.User, ctx.UID)},
		{"CPU", ctx.Hardware.CPUModel},
		{"Cores", fmt.Sprintf("%d", ctx.Hardware.CPUCore)},
		{"RAM", ctx.Hardware.RAMTotal},
		{"GPU", fmt.Sprintf("%s %s", ctx.Hardware.GPUVendor, ctx.Hardware.GPUModel)},
		{"Shell", ctx.Env.Shell},
		{"FS", ctx.FS.RootFSType},
		{"Time", time.Now().Format("15:04:05")},
	}
	pterm.DefaultTable.WithHasHeader(false).WithData(sysInfo).Render()
	pterm.Println()

	// 2. Initialize State Manager
	statePath := filepath.Join(".veto", "state.json")
	stateMgr, err := state.NewManager(statePath)
	if err != nil {
		pterm.Warning.Printf("Could not initialize state manager: %v\n", err)
	}

	// 3. Load Configuration
	spinnerLoad, _ := pterm.DefaultSpinner.Start("Loading configuration...")
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		spinnerLoad.Fail(fmt.Sprintf("Error loading config file '%s': %v", configFile, err))
		return err
	}
	spinnerLoad.Success("Configuration loaded")

	// 4. Sort Resources
	spinnerSort, _ := pterm.DefaultSpinner.Start("Resolving dependencies...")
	sortedResources, err := config.SortResources(cfg.Resources)
	if err != nil {
		spinnerSort.Fail(fmt.Sprintf("Error sorting resources: %v", err))
		return err
	}
	spinnerSort.Success(fmt.Sprintf("Resolved %d layers", len(sortedResources)))
	pterm.Println()

	// 5. Prepare Engine
	eng := core.NewEngine(ctx, stateMgr)

	// 6. Fire Engine
	createFn := func(t, n string, p map[string]interface{}, c *core.SystemContext) (core.Resource, error) {
		return resource.CreateResourceWithParams(t, n, p, c)
	}

	finalError := error(nil)

	for i, layer := range sortedResources {
		// Layer Header
		pterm.DefaultSection.Printf("Phase %d: Processing %d resources", i+1, len(layer))

		// Resources...
		var layerItems []core.ConfigItem
		for _, res := range layer {
			name := res.Name
			if name == "" {
				if n, ok := res.Params["name"].(string); ok {
					name = n
				}
			}
			if name == "" {
				name = res.ID
			}
			state := res.State
			if state == "" {
				if s, ok := res.Params["state"].(string); ok {
					state = s
				}
			}
			layerItems = append(layerItems, core.ConfigItem{
				Name:   name,
				Type:   res.Type,
				State:  state,
				When:   res.When,
				Params: res.Params,
			})
		}

		// Spinner for execution (Simple main spinner)
		spinnerExec, _ := pterm.DefaultSpinner.Start("Executing layer...")

		if err := eng.RunParallel(layerItems, createFn); err != nil {
			spinnerExec.Fail(fmt.Sprintf("Layer %d failed", i+1))
			pterm.Error.Printf("Layer %d completed with errors: %v\n", i+1, err)
			finalError = err // Keep track of error
			break            // Stop processing layers on failure
		}
		spinnerExec.Success(fmt.Sprintf("Layer %d complete", i+1))
	}

	// Post Snapshot Logic
	if preSnapID != "" && snapMgr != nil {
		desc := fmt.Sprintf("Post-Veto Apply: %s", configFile)
		if finalError != nil {
			desc += " (Failed)"
		}

		// Manager arka tarafta Timeshift ise post-snapshot'ı atlayabilir
		if err := snapMgr.CreatePostSnapshot(preSnapID, desc); err != nil {
			pterm.Warning.Printf("Post-snapshot failed: %v\n", err)
		}
	}

	if finalError != nil {
		return finalError
	}

	// 7. Prune Logic (Strict Mode)
	if isPrune {
		pterm.Println()
		pterm.DefaultHeader.WithFullWidth().
			WithBackgroundStyle(pterm.NewStyle(pterm.BgRed)).
			WithTextStyle(pterm.NewStyle(pterm.FgWhite, pterm.Bold)).
			Println("PRUNE MODE (Destructive)")

		// Convert all resources to ConfigItems for Prune
		var allItems []core.ConfigItem
		for _, res := range cfg.Resources {
			allItems = append(allItems, core.ConfigItem{
				Name: res.Name,
				Type: res.Type,
			})
		}

		if err := eng.Prune(allItems, createFn); err != nil {
			pterm.Error.Printf("Prune failed: %v\n", err)
			return err
		}
	}

	pterm.Println()
	pterm.DefaultBasicText.WithStyle(pterm.NewStyle(pterm.FgGreen, pterm.Bold)).Println("✨ Configuration applied successfully!")
	return nil
}
