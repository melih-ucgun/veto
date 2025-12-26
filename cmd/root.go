package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "monarch",
	Short: "Your System, Your Rules. Enforced by Monarch.",
	Long:  `Monarch is a declarative, agentless configuration management tool.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringP("config", "c", "monarch.yaml", "config file path")
}
