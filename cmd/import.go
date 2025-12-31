package cmd

import (
	"fmt"
	"os"

	"github.com/melih-ucgun/monarch/internal/discovery"
	"github.com/melih-ucgun/monarch/internal/system"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var importCmd = &cobra.Command{
	Use:   "import [output_file]",
	Short: "Discover installed packages and services",
	Long:  `Scans the system for explicitly installed packages and enabled services, and generates a Monarch configuration file.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		outputFile := "imported_system.yaml"
		if len(args) > 0 {
			outputFile = args[0]
		}

		pterm.DefaultHeader.Println("System Discovery & Import")
		spinner, _ := pterm.DefaultSpinner.Start("Detecting system context...")

		ctx := system.Detect(false)

		spinner.UpdateText("Discovering packages and services...")
		cfg, err := discovery.DiscoverSystem(ctx)
		if err != nil {
			spinner.Fail("Discovery failed: " + err.Error())
			return
		}

		spinner.Success(fmt.Sprintf("Discovery complete. Found %d resources.", len(cfg.Resources)))

		// Marshal to YAML
		data, err := yaml.Marshal(cfg)
		if err != nil {
			pterm.Error.Println("Failed to marshal config:", err)
			return
		}

		// Write to file
		if err := os.WriteFile(outputFile, data, 0644); err != nil {
			pterm.Error.Println("Failed to write output file:", err)
			return
		}

		pterm.Success.Printf("Configuration saved to %s\n", outputFile)
		pterm.Info.Println("Review this file before running 'monarch apply'!")
	},
}

func init() {
	rootCmd.AddCommand(importCmd)
}
