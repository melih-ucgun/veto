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
	State  *State // State eklendi
}

func NewReconciler(cfg *config.Config, opts EngineOptions) *Reconciler {
	// Engine başlarken state'i yükle
	state, _ := LoadState()
	return &Reconciler{
		Config: cfg,
		Opts:   opts,
		State:  state,
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
			diff, _ := res.Diff()

			if e.Opts.DryRun {
				slog.Info("DRY-RUN: Sapma tespit edildi", "id", res.ID())
				if diff != "" {
					slog.Info("Planlanan değişiklik", "diff", diff)
				}
			} else {
				if e.Opts.AutoHeal || !isWatchContext(e.Opts) {
					slog.Info("Değişiklik uygulanıyor", "id", res.ID())

					applyErr := res.Apply()

					// State güncelleme (başarılı veya başarısız fark etmez, denendiğini kaydediyoruz)
					if e.State != nil {
						e.State.UpdateResource(res.ID(), r.Type, applyErr == nil)
					}

					if applyErr != nil {
						slog.Error("Uygulama hatası", "id", res.ID(), "error", applyErr)
					} else {
						slog.Info("Başarıyla uygulandı", "id", res.ID())
					}
				} else {
					slog.Warn("SAPMA TESPİT EDİLDİ", "id", res.ID())
				}
			}
		}
	}

	// Tüm işlemler bitince state dosyasını kaydet
	if !e.Opts.DryRun && e.State != nil {
		if err := e.State.Save(); err != nil {
			slog.Error("State dosyası kaydedilemedi", "error", err)
		} else {
			slog.Debug("Sistem durumu kaydedildi.")
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
