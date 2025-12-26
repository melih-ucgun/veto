package cmd

import (
	"fmt"
	"os"

	"github.com/melih-ucgun/monarch/internal/config" // tam yol bu
	"github.com/spf13/cobra"
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply the desired state",
	Run: func(cmd *cobra.Command, args []string) {
		configFile, _ := rootCmd.PersistentFlags().GetString("config")

		cfg, err := config.LoadConfig(configFile)
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("🏰 Monarch apply started!")
		fmt.Printf("Loaded config: %s\n", configFile)
		fmt.Printf("Found %d resource(s)\n", len(cfg.Resources))

		for _, r := range cfg.Resources {
			fmt.Printf(" - %s: %s\n", r.Type, r.Name)
		}
	},
}

func init() {
	rootCmd.AddCommand(applyCmd)
}
