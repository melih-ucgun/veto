package cmd

import (
	"log/slog"
	"os"

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
	// Varsayılan JSON loglayıcı ayarla (veya isteğe bağlı TextHandler)
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(handler))

	rootCmd.PersistentFlags().StringP("config", "c", "monarch.yaml", "config file path")
}
