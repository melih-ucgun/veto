package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var interval int
var withSnapshot bool

var watchCmd = &cobra.Command{
	Use:   "watch [config_file]",
	Short: "Watch for changes in the configuration file and apply automatically",
	Long: `Monitors the specified configuration file (default: veto.yaml) for changes.
When a change is detected, it automatically runs 'apply'.
Polls the file system every few seconds (configurable).`,
	Run: func(cmd *cobra.Command, args []string) {
		configFile := "veto.yaml"
		if len(args) > 0 {
			configFile = args[0]
		}

		fmt.Printf("👀 Watching '%s' for changes (Interval: %ds)...\n", configFile, interval)
		fmt.Println("Press Ctrl+C to stop.")

		// İlk başlangıçta bir kez çalıştır
		// İlk başlangıçta bir kez çalıştır
		// Watch mode runs locally, so inventory is empty string and concurrency is not used (pass 1).
		if err := runApply(configFile, "", 1, dryRun, !withSnapshot, false); err != nil {
			fmt.Printf("⚠️ Initial apply failed, but keeping watch...\n")
		}

		watchLoop(configFile, interval)
	},
}

func init() {
	rootCmd.AddCommand(watchCmd)
	watchCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Simulate changes without applying them")
	watchCmd.Flags().IntVarP(&interval, "interval", "i", 2, "Polling interval in seconds")
	watchCmd.Flags().BoolVar(&withSnapshot, "with-snapshot", false, "Enable automatic snapshots during watch")
}

func watchLoop(filename string, intervalSec int) {
	lastModTime := time.Time{}

	// İlk dosya bilgisini al
	info, err := os.Stat(filename)
	if err == nil {
		lastModTime = info.ModTime()
	}

	ticker := time.NewTicker(time.Duration(intervalSec) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		info, err := os.Stat(filename)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("\r⚠️ Config file '%s' not found. Waiting...", filename)
			}
			continue
		}

		// Değişiklik kontrolü
		if info.ModTime().After(lastModTime) {
			fmt.Println("\n\n🔄 Change detected! Re-applying configuration...")

			// Güncel zamanı kaydet
			lastModTime = info.ModTime()

			// Apply işlemini çağır (cmd/apply.go içindeki fonksiyonu kullanıyoruz)
			// Not: runApply fonksiyonu aynı pakette (cmd) olduğu için erişilebilir.
			if err := runApply(filename, "", 1, dryRun, !withSnapshot, false); err != nil {
				fmt.Printf("❌ Apply failed: %v\n", err)
			} else {
				fmt.Printf("✅ Update successful. Watching for new changes...\n")
			}
		}
	}
}
