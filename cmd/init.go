package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"atomicgo.dev/cursor"
	"github.com/melih-ucgun/veto/internal/core"
	"github.com/melih-ucgun/veto/internal/system"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var autoConfirm bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Veto system profile",
	Long:  `Scans the current system and creates a '.veto/system.yaml' profile used for context-aware operations.`,
	Run: func(cmd *cobra.Command, args []string) {
		runInit()
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolVarP(&autoConfirm, "yes", "y", false, "Skip interactive prompts and save immediately")
}

func runInit() {
	pterm.DefaultHeader.WithFullWidth().WithBackgroundStyle(pterm.NewStyle(pterm.BgMagenta)).
		WithTextStyle(pterm.NewStyle(pterm.FgBlack, pterm.Bold)).
		Println("Veto System Initializer")

	spinner, _ := pterm.DefaultSpinner.Start("Scanning system...")
	detectedCtx := system.Detect(false)
	spinner.Success("System scan complete")
	pterm.Println()

	// Show Results
	displaySystemInfo(detectedCtx)

	if autoConfirm {
		pterm.Info.Println("Auto-confirm enabled. Saving profile...")
	} else {
		// Interaction Loop
		selection, _ := pterm.DefaultInteractiveSelect.
			WithOptions([]string{"Yes, looks good", "Customize", "Cancel"}).
			Show("Do you want to save this system profile?")

		if selection == "Cancel" {
			pterm.Info.Println("Initialization cancelled.")
			return
		}

		if selection == "Customize" {
			customizeProfile(detectedCtx)
			pterm.Println()
			pterm.Info.Println("Updated Profile:")
			displaySystemInfo(detectedCtx)
		}
	}

	// Save
	if err := saveSystemProfile(detectedCtx); err != nil {
		pterm.Error.Printf("Failed to save profile: %v\n", err)
		os.Exit(1)
	}

	pterm.Success.Println("System profile saved to .veto/system.yaml")
	pterm.Info.Println("You can now run 'veto apply' and it will use this context!")
}

func displaySystemInfo(ctx *core.SystemContext) {
	data := [][]string{
		{"OS", ctx.OS},
		{"Distro", ctx.Distro},
		{"Version", ctx.Version},
		{"Hostname", ctx.Hostname},
		{"User", ctx.User},
		{"Shell", ctx.Env.Shell},
		{"CPU", ctx.Hardware.CPUModel},
		{"GPU", ctx.Hardware.GPUVendor},
		{"FS", ctx.FS.RootFSType},
	}
	pterm.DefaultTable.WithHasHeader(false).WithData(data).Render()
}

func customizeProfile(ctx *core.SystemContext) {
	cursor.Show()
	pterm.Info.Println("Enter new values (leave empty to keep current):")

	ctx.Distro = ask("Distro", ctx.Distro)
	ctx.Version = ask("Version", ctx.Version)
	ctx.Env.Shell = ask("Shell", ctx.Env.Shell)
	ctx.Hardware.GPUVendor = ask("GPU Vendor", ctx.Hardware.GPUVendor)

	cursor.Hide()
}

func ask(label, current string) string {
	prompt := fmt.Sprintf("%s [%s]: ", label, current)
	fmt.Print(prompt)
	var input string
	fmt.Scanln(&input)
	if input == "" {
		return current
	}
	return input
}

func saveSystemProfile(ctx *core.SystemContext) error {
	// Create .veto dir if not exists
	vetoDir := ".veto"
	if err := os.MkdirAll(vetoDir, 0755); err != nil {
		return err
	}

	filePath := filepath.Join(vetoDir, "system.yaml")
	data, err := yaml.Marshal(ctx)
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}
