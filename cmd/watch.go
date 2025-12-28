package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/melih-ucgun/monarch/internal/config"
	"github.com/melih-ucgun/monarch/internal/resources"
	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Sistemi sürekli gözlemler ve sapmaları raporlar",
	Long:  `Konfigürasyon dosyasını periyodik olarak kontrol eder. Eğer sistemde bir sapma (drift) bulursa sizi uyarır veya otomatik düzeltir.`,
	Run: func(cmd *cobra.Command, args []string) {
		configFile, _ := rootCmd.PersistentFlags().GetString("config")
		interval, _ := cmd.Flags().GetInt("interval")
		autoHeal, _ := cmd.Flags().GetBool("auto-heal")

		fmt.Printf("👁️ Monarch Watch başlatıldı. (Aralık: %d saniye, Otomatik Düzeltme: %v)\n", interval, autoHeal)
		fmt.Println("Durdurmak için Ctrl+C tuşlarına basın.")

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		ticker := time.NewTicker(time.Duration(interval) * time.Second)
		defer ticker.Stop()

		runWatchCycle(configFile, autoHeal)

		for {
			select {
			case <-ticker.C:
				runWatchCycle(configFile, autoHeal)
			case <-sigChan:
				fmt.Println("\n👋 Monarch Watch durduruluyor...")
				return
			}
		}
	},
}

func runWatchCycle(configFile string, autoHeal bool) {
	fmt.Printf("[%s] 🔍 Kontrol ediliyor...\n", time.Now().Format("15:04:05"))

	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		fmt.Printf("❌ Konfigürasyon hatası: %v\n", err)
		return
	}

	sortedResources, err := config.SortResources(cfg.Resources)
	if err != nil {
		fmt.Printf("❌ Sıralama hatası: %v\n", err)
		return
	}

	driftsFound := 0

	for _, r := range sortedResources {
		// apply.go ile aynı fabrika metodunu kullanıyoruz
		res, err := resources.New(r, cfg.Vars)
		if err != nil || res == nil {
			continue
		}

		isInState, err := res.Check()
		if err != nil {
			continue
		}

		if !isInState {
			driftsFound++
			fmt.Printf("⚠️  SAPMA TESPİT EDİLDİ: [%s]\n", res.ID())

			if autoHeal {
				fmt.Printf("   🛠️  Otomatik düzeltiliyor...\n")
				if err := res.Apply(); err != nil {
					fmt.Printf("   ❌ Düzeltme hatası: %v\n", err)
				} else {
					fmt.Printf("   ✨ Düzeldi!\n")
				}
			}
		}
	}

	if driftsFound > 0 && !autoHeal {
		fmt.Printf("📢 Toplam %d sapma bulundu. Düzelmek için 'monarch apply' kullanın.\n", driftsFound)
	}
}

func init() {
	rootCmd.AddCommand(watchCmd)
	watchCmd.Flags().IntP("interval", "i", 30, "Kontrol aralığı (saniye)")
	watchCmd.Flags().BoolP("auto-heal", "a", false, "Sapmaları otomatik olarak düzelt")
}
