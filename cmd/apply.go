package cmd

import (
	"fmt"
	"os"

	"github.com/melih-ucgun/monarch/internal/config"
	"github.com/melih-ucgun/monarch/internal/resources"
	"github.com/spf13/cobra"
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply the desired state to the system",
	Run: func(cmd *cobra.Command, args []string) {
		configFile, _ := rootCmd.PersistentFlags().GetString("config")
		// 1. Dry-run bayrağını oku
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		// 2. Yapılandırmayı Yükle
		cfg, err := config.LoadConfig(configFile)
		if err != nil {
			fmt.Printf("❌ Error loading config: %v\n", err)
			os.Exit(1)
		}

		// 3. Kaynakları Bağımlılıklara Göre Sırala
		sortedResources, err := config.SortResources(cfg.Resources)
		if err != nil {
			fmt.Printf("❌ Dependency Error: %v\n", err)
			os.Exit(1)
		}

		if dryRun {
			fmt.Println("🔍 [DRY-RUN MODE] No changes will be actually applied to your system.")
		}

		fmt.Println("🏰 Monarch is ensuring your sovereignty...")
		fmt.Printf("📂 Using config: %s\n", configFile)
		fmt.Printf("🔍 Found %d resource(s) to check\n\n", len(sortedResources))

		// 4. Sıralanmış kaynakları döngüye al
		for _, r := range sortedResources {

			// Şablon İşleme (Templating)
			processedContent := r.Content
			if r.Content != "" {
				var err error
				processedContent, err = config.ExecuteTemplate(r.Content, cfg.Vars)
				if err != nil {
					fmt.Printf("❌ [%s] Template processing failed: %v\n", r.Name, err)
					continue
				}
			}

			var res resources.Resource

			// Kaynak nesnesini oluştur
			switch r.Type {
			case "file":
				res = &resources.FileResource{
					ResourceName: r.Name,
					Path:         r.Path,
					Content:      processedContent,
				}
			case "package":
				res = &resources.PackageResource{
					PackageName: r.Name,
					State:       r.State,
					Provider:    resources.GetDefaultProvider(),
				}
			case "service":
				res = &resources.ServiceResource{
					ServiceName:  r.Name,
					DesiredState: r.State,
					Enabled:      r.Enabled,
				}
			case "noop":
				fmt.Printf("ℹ️ Skipping noop resource: %s\n", r.Name)
				continue
			default:
				fmt.Printf("⚠️ Unknown resource type: %s (Name: %s)\n", r.Type, r.Name)
				continue
			}

			// 5. Durum Kontrolü
			isInState, err := res.Check()
			if err != nil {
				fmt.Printf("❌ [%s] Check failed: %v\n", res.ID(), err)
				continue
			}

			if isInState {
				fmt.Printf("✅ [%s] is already in the desired state.\n", res.ID())
			} else {
				// 6. Eğer dry-run aktifse uygulama, sadece bilgi ver
				if dryRun {
					fmt.Printf("🔍 [DRY-RUN] [%s] is out of sync. Change would be applied.\n", res.ID())
				} else {
					fmt.Printf("🛠️ [%s] is out of sync. Applying changes...\n", res.ID())
					if err := res.Apply(); err != nil {
						fmt.Printf("❌ [%s] Apply failed: %v\n", res.ID(), err)
					} else {
						fmt.Printf("✨ [%s] successfully applied!\n", res.ID())
					}
				}
			}
		}

		if dryRun {
			fmt.Println("\n🏁 Monarch dry-run finished. No system changes were made.")
		} else {
			fmt.Println("\n🏁 Monarch apply finished.")
		}
	},
}

func init() {
	rootCmd.AddCommand(applyCmd)
	// Dry-run flag tanımı
	applyCmd.Flags().BoolP("dry-run", "d", false, "Don't apply changes, only show what would be done")
	applyCmd.Flags().StringP("host", "H", "localhost", "Target host for apply")
}
