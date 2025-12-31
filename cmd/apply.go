package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/melih-ucgun/monarch/internal/config"
	"github.com/melih-ucgun/monarch/internal/core"
	"github.com/melih-ucgun/monarch/internal/resource"
	"github.com/melih-ucgun/monarch/internal/state" // New import
	"github.com/melih-ucgun/monarch/internal/system"
)

var dryRun bool

var applyCmd = &cobra.Command{
	Use:   "apply [config_file]",
	Short: "Apply the configuration to the system",
	Long: `Reads the configuration file and ensures system state matches desired state.
Updates .monarch/state.json with the results.`,
	Run: func(cmd *cobra.Command, args []string) {
		configFile := "monarch.yaml"
		if len(args) > 0 {
			configFile = args[0]
		}

		if err := runApply(configFile, dryRun); err != nil {
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(applyCmd)
	applyCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Simulate changes without applying them")
}

func runApply(configFile string, isDryRun bool) error {
	// Header
	pterm.DefaultHeader.WithFullWidth().WithBackgroundStyle(pterm.NewStyle(pterm.BgLightBlue)).
		WithTextStyle(pterm.NewStyle(pterm.FgBlack, pterm.Bold)).
		Println("Monarch Config Manager")

	if isDryRun {
		pterm.ThemeDefault.SecondaryStyle.Println("Running in DRY-RUN mode")
	}

	// 1. Detect System
	ctx := system.Detect(isDryRun)

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
	statePath := filepath.Join(".monarch", "state.json")
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
	createFn := func(t, n string, p map[string]interface{}, c *core.SystemContext) (core.ApplyableResource, error) {
		return resource.CreateResourceWithParams(t, n, p, c)
	}

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
			return err
		}
		spinnerExec.Success(fmt.Sprintf("Layer %d complete", i+1))
	}

	pterm.Println()
	pterm.DefaultBasicText.WithStyle(pterm.NewStyle(pterm.FgGreen, pterm.Bold)).Println("✨ Configuration applied successfully!")
	return nil
}
