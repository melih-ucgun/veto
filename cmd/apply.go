package cmd

import (
	"fmt"
	"os"

	"github.com/melih-ucgun/monarch/internal/config"
	"github.com/melih-ucgun/monarch/internal/resources"
	"github.com/melih-ucgun/monarch/internal/transport"
	"github.com/spf13/cobra"
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Sistemi arzu edilen duruma getirir",
	Run: func(cmd *cobra.Command, args []string) {
		configFile, _ := rootCmd.PersistentFlags().GetString("config")
		hostName, _ := cmd.Flags().GetString("host")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		cfg, err := config.LoadConfig(configFile)
		if err != nil {
			fmt.Printf("❌ Konfigürasyon yüklenemedi: %v\n", err)
			os.Exit(1)
		}

		if hostName != "localhost" {
			executeRemote(hostName, configFile, dryRun, cfg)
			return
		}

		executeLocal(configFile, dryRun, cfg)
	},
}

func executeRemote(hostName, configFile string, dryRun bool, cfg *config.Config) {
	fmt.Printf("🌐 Uzak sunucuya bağlanılıyor: %s\n", hostName)

	var targetHost *config.Host
	for _, h := range cfg.Hosts {
		if h.Name == hostName {
			targetHost = &h
			break
		}
	}

	if targetHost == nil {
		fmt.Printf("❌ Hata: '%s' isimli host konfigürasyon dosyasında bulunamadı.\n", hostName)
		os.Exit(1)
	}

	t, err := transport.NewSSHTransport(*targetHost)
	if err != nil {
		fmt.Printf("❌ SSH bağlantısı kurulamadı: %v\n", err)
		os.Exit(1)
	}

	selfPath, err := os.Executable()
	if err != nil {
		fmt.Printf("❌ Kendi executable dosyası bulunamadı: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("🚀 Monarch binary dosyası uzak sunucuya kopyalanıyor...")
	if err := t.CopyFile(selfPath, "/tmp/monarch"); err != nil {
		fmt.Printf("❌ Binary kopyalanamadı: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("🚀 Konfigürasyon dosyası uzak sunucuya kopyalanıyor...")
	if err := t.CopyFile(configFile, "/tmp/monarch.yaml"); err != nil {
		fmt.Printf("❌ Konfigürasyon kopyalanamadı: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("🏰 Uzak sunucuda Monarch başlatılıyor...")
	remoteCmd := "chmod +x /tmp/monarch && sudo /tmp/monarch apply --config /tmp/monarch.yaml"
	if dryRun {
		remoteCmd += " --dry-run"
	}

	if err := t.RunRemote(remoteCmd); err != nil {
		fmt.Printf("❌ Uzak çalıştırma başarısız oldu: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n🏁 Uzak sunucu işlemi tamamlandı.")
}

func executeLocal(configFile string, dryRun bool, cfg *config.Config) {
	sortedResources, err := config.SortResources(cfg.Resources)
	if err != nil {
		fmt.Printf("❌ Bağımlılık Hatası: %v\n", err)
		os.Exit(1)
	}

	if dryRun {
		fmt.Println("🔍 [DRY-RUN MODU] Sisteme gerçek bir değişiklik uygulanmayacak.")
	}

	fmt.Println("🏰 Monarch sisteminize hükmediyor...")
	fmt.Printf("📂 Kullanılan dosya: %s\n", configFile)
	fmt.Printf("🔍 %d kaynak kontrol edilecek\n\n", len(sortedResources))

	for _, r := range sortedResources {
		// Fabrikayı (Factory) kullanarak kaynağı oluşturuyoruz
		res, err := resources.New(r, cfg.Vars)
		if err != nil {
			fmt.Printf("⚠️ [%s] Kaynak oluşturma hatası: %v\n", r.Name, err)
			continue
		}

		// noop tipi nil döndüğü için kontrol edip atlıyoruz
		if res == nil {
			fmt.Printf("ℹ️ [%s] atlanıyor (tip: %s)\n", r.Name, r.Type)
			continue
		}

		isInState, err := res.Check()
		if err != nil {
			fmt.Printf("❌ [%s] Kontrol başarısız: %v\n", res.ID(), err)
			continue
		}

		if isInState {
			fmt.Printf("✅ [%s] zaten istenen durumda.\n", res.ID())
		} else {
			if dryRun {
				fmt.Printf("🔍 [DRY-RUN] [%s] senkronize değil. Değişiklik uygulanabilir.\n", res.ID())
			} else {
				fmt.Printf("🛠️ [%s] senkronize değil. Uygulanıyor...\n", res.ID())
				if err := res.Apply(); err != nil {
					fmt.Printf("❌ [%s] Uygulama hatası: %v\n", res.ID(), err)
				} else {
					fmt.Printf("✨ [%s] başarıyla uygulandı!\n", res.ID())
				}
			}
		}
	}

	fmt.Println("\n🏁 Monarch işlemi tamamladı.")
}

func init() {
	rootCmd.AddCommand(applyCmd)
	applyCmd.Flags().BoolP("dry-run", "d", false, "Değişiklikleri uygulama, sadece ne yapılacağını göster")
	applyCmd.Flags().StringP("host", "H", "localhost", "Hedef sunucu (config dosyasındaki host adı)")
}
