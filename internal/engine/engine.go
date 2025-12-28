package engine

import (
	"log/slog"
	"time"

	"github.com/melih-ucgun/monarch/internal/config"
	"github.com/melih-ucgun/monarch/internal/resources"
)

type EngineOptions struct {
	DryRun   bool
	AutoHeal bool
}

type Reconciler struct {
	Config *config.Config
	Opts   EngineOptions
}

func NewReconciler(cfg *config.Config, opts EngineOptions) *Reconciler {
	return &Reconciler{
		Config: cfg,
		Opts:   opts,
	}
}

func (e *Reconciler) Run() (int, error) {
	sortedResources, err := config.SortResources(e.Config.Resources)
	if err != nil {
		return 0, err
	}

	driftsFound := 0

	for _, r := range sortedResources {
		res, err := resources.New(r, e.Config.Vars)
		if err != nil {
			slog.Error("Kaynak oluşturma hatası", "resource", r.Name, "error", err)
			continue
		}

		if res == nil {
			continue
		}

		isInState, err := res.Check()
		if err != nil {
			slog.Error("Kontrol başarısız", "id", res.ID(), "error", err)
			continue
		}

		if isInState {
			if !e.Opts.AutoHeal {
				slog.Info("Kaynak istenen durumda", "id", res.ID())
			}
		} else {
			driftsFound++

			// Değişiklik farkını al
			diff, _ := res.Diff()

			if e.Opts.DryRun {
				slog.Info("DRY-RUN: Sapma tespit edildi", "id", res.ID())
				if diff != "" {
					// Diff çıktısını loga ekle
					slog.Info("Planlanan değişiklik", "diff", diff)
				}
			} else {
				if e.Opts.AutoHeal || !isWatchContext(e.Opts) {
					slog.Info("Değişiklik uygulanıyor", "id", res.ID())
					if diff != "" {
						slog.Info("Fark", "content", diff)
					}

					if err := res.Apply(); err != nil {
						slog.Error("Uygulama hatası", "id", res.ID(), "error", err)
					} else {
						slog.Info("Başarıyla uygulandı", "id", res.ID())
					}
				} else {
					slog.Warn("SAPMA TESPİT EDİLDİ", "id", res.ID())
				}
			}
		}
	}

	return driftsFound, nil
}

func isWatchContext(opts EngineOptions) bool {
	return opts.AutoHeal
}

func LogTimestamp(msg string) {
	slog.Info(msg, "time", time.Now().Format("15:04:05"))
}
