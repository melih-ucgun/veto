package engine

import (
	"fmt"
	"time"

	"github.com/melih-ucgun/monarch/internal/config"
	"github.com/melih-ucgun/monarch/internal/resources"
)

// EngineOptions, engine'in çalışma şeklini belirleyen parametrelerdir.
type EngineOptions struct {
	DryRun   bool // Gerçek değişiklik yapmadan sadece simüle eder
	AutoHeal bool // Sapma tespit edildiğinde otomatik düzeltir
}

// Reconciler, sistemin arzu edilen durumu ile mevcut durumu arasındaki dengeyi sağlar.
type Reconciler struct {
	Config *config.Config
	Opts   EngineOptions
}

// NewReconciler, yeni bir engine (uzlaştırıcı) örneği oluşturur.
func NewReconciler(cfg *config.Config, opts EngineOptions) *Reconciler {
	return &Reconciler{
		Config: cfg,
		Opts:   opts,
	}
}

// Run, tüm kaynakları bağımlılık sırasına göre işler ve sistem durumunu eşitler.
// Geriye bulunan toplam sapma (drift) sayısını döndürür.
func (e *Reconciler) Run() (int, error) {
	// 1. Kaynakları bağımlılıklarına (DependsOn) göre topolojik olarak sırala
	sortedResources, err := config.SortResources(e.Config.Resources)
	if err != nil {
		return 0, fmt.Errorf("bağımlılık sıralama hatası: %w", err)
	}

	driftsFound := 0

	for _, r := range sortedResources {
		// 2. Resource nesnesini (File, Package, Service vb.) factory üzerinden oluştur
		res, err := resources.New(r, e.Config.Vars)
		if err != nil {
			fmt.Printf("⚠️ [%s] Kaynak oluşturma hatası: %v\n", r.Name, err)
			continue
		}

		// noop veya bilinmeyen tipler nil dönebilir, güvenli çıkış yapalım
		if res == nil {
			continue
		}

		// 3. Mevcut durumun (Actual) arzu edilen durumla (Desired) uyumunu kontrol et
		isInState, err := res.Check()
		if err != nil {
			fmt.Printf("❌ [%s] Kontrol başarısız: %v\n", res.ID(), err)
			continue
		}

		if isInState {
			// Eğer her şey yolundaysa ve watch modunda değilsek bilgi verelim
			if !e.Opts.AutoHeal {
				fmt.Printf("✅ [%s] zaten istenen durumda.\n", res.ID())
			}
		} else {
			// Sapma bulundu
			driftsFound++

			if e.Opts.DryRun {
				fmt.Printf("🔍 [DRY-RUN] [%s] senkronize değil. Değişiklik uygulanabilir.\n", res.ID())
			} else {
				// Uygulama (Apply) kararı: Ya watch mode'da auto-heal açıktır, ya da direkt apply komutu çalışıyordur.
				if e.Opts.AutoHeal || !isWatchContext(e.Opts) {
					fmt.Printf("🛠️ [%s] senkronize değil. Uygulanıyor...\n", res.ID())
					if err := res.Apply(); err != nil {
						fmt.Printf("❌ [%s] Uygulama hatası: %v\n", res.ID(), err)
					} else {
						fmt.Printf("✨ [%s] başarıyla uygulandı!\n", res.ID())
					}
				} else {
					// Watch modundayız ama auto-heal kapalıysa sadece uyarı veriyoruz
					fmt.Printf("⚠️ SAPMA TESPİT EDİLDİ: [%s]\n", res.ID())
				}
			}
		}
	}

	return driftsFound, nil
}

// isWatchContext, mevcut ayarların bir 'izleme' (watch) senaryosuna ait olup olmadığını kontrol eder.
func isWatchContext(opts EngineOptions) bool {
	// AutoHeal bayrağı sadece watch komutu bağlamında anlamlıdır.
	return opts.AutoHeal
}

// LogTimestamp, zaman damgalı çıktı üretmek için yardımcı fonksiyondur.
func LogTimestamp(msg string) {
	fmt.Printf("[%s] %s\n", time.Now().Format("15:04:05"), msg)
}
