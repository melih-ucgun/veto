package engine

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/melih-ucgun/monarch/internal/config"
	"github.com/melih-ucgun/monarch/internal/resources"
	"github.com/melih-ucgun/monarch/internal/transport"
	"golang.org/x/sync/errgroup"
)

//go:embed embedded/*
var embeddedBinaries embed.FS

type EngineOptions struct {
	DryRun     bool
	AutoHeal   bool
	HostName   string
	ConfigFile string
}

type Reconciler struct {
	Config     *config.Config
	Opts       EngineOptions
	State      *State
	stateMutex sync.Mutex
}

func NewReconciler(cfg *config.Config, opts EngineOptions) *Reconciler {
	state, _ := LoadState()
	return &Reconciler{Config: cfg, Opts: opts, State: state}
}

// Run context alır ve dağıtır
func (e *Reconciler) Run(ctx context.Context) (int, error) {
	if e.Opts.HostName == "" || e.Opts.HostName == "localhost" {
		return e.runLocal(ctx)
	}
	return e.runRemote(ctx)
}

func (e *Reconciler) runLocal(ctx context.Context) (int, error) {
	levels, err := config.SortResources(e.Config.Resources)
	if err != nil {
		return 0, fmt.Errorf("sıralama hatası: %w", err)
	}

	drifts := 0
	var driftsMutex sync.Mutex

	for i, level := range levels {
		if ctx.Err() != nil {
			return drifts, ctx.Err()
		}

		slog.Debug("Katman işleniyor", "seviye", i+1, "kaynak_sayisi", len(level))

		g, _ := errgroup.WithContext(ctx)

		for _, rCfg := range level {
			currentRCfg := rCfg
			g.Go(func() error {
				if ctx.Err() != nil {
					return ctx.Err()
				}

				res, err := resources.New(currentRCfg, e.Config.Vars)
				if err != nil {
					return err
				}

				ok, err := res.Check()
				if err != nil {
					return fmt.Errorf("check hatası [%s]: %w", res.ID(), err)
				}

				if !ok {
					driftsMutex.Lock()
					drifts++
					driftsMutex.Unlock()

					diff, _ := res.Diff()
					if e.Opts.DryRun {
						slog.Info("SAPMA (Dry-Run)", "id", res.ID(), "diff", diff)
					} else {
						slog.Info("Uygulanıyor", "id", res.ID())
						if applyErr := res.Apply(); applyErr != nil {
							return fmt.Errorf("apply hatası [%s]: %w", res.ID(), applyErr)
						}

						if e.State != nil {
							e.stateMutex.Lock()
							e.State.UpdateResource(res.ID(), currentRCfg.Type, true)
							e.stateMutex.Unlock()
						}
					}
				}
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			return drifts, err
		}
	}

	if !e.Opts.DryRun && e.State != nil {
		_ = e.State.Save()
	}
	return drifts, nil
}

func (e *Reconciler) runRemote(ctx context.Context) (int, error) {
	var target *config.Host
	for i := range e.Config.Hosts {
		if e.Config.Hosts[i].Name == e.Opts.HostName {
			target = &e.Config.Hosts[i]
			break
		}
	}
	if target == nil {
		return 0, fmt.Errorf("host bulunamadı: %s", e.Opts.HostName)
	}

	// Yerel konfigürasyon dosyasını belleğe oku
	configContent, err := os.ReadFile(e.Opts.ConfigFile)
	if err != nil {
		return 0, fmt.Errorf("konfig dosyası okunamadı: %w", err)
	}

	t, err := transport.NewSSHTransport(ctx, *target)
	if err != nil {
		return 0, err
	}
	defer t.Close()

	remoteOS, remoteArch, err := t.GetRemoteSystemInfo(ctx)
	if err != nil {
		return 0, err
	}

	// Binary yolunu çözümle (Embed edilmiş dosyalardan çıkar)
	binaryPath, err := resolveBinaryPath(remoteOS, remoteArch)
	if err != nil {
		return 0, err
	}
	// Geçici dosyayı iş bitince temizlemek için defer (Local temizlik)
	defer os.Remove(binaryPath)

	timestamp := time.Now().Format("20060102150405")
	remoteBinPath := fmt.Sprintf("/tmp/monarch-%s", timestamp)

	// Binary'yi kopyala
	slog.Info("Binary uzak sunucuya gönderiliyor...", "local_path", binaryPath, "remote_os", remoteOS)
	if err := t.CopyFile(ctx, binaryPath, remoteBinPath); err != nil {
		return 0, err
	}

	// --- TEMİZLİK (REMOTE CLEANUP) ---
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Sadece binary'i siliyoruz
		err := t.RunRemoteSecure(cleanupCtx, fmt.Sprintf("rm -f %s", remoteBinPath), "", "")
		if err != nil {
			slog.Warn("Temizlik sırasında hata oluştu", "error", err)
		}
	}()
	// ---------------------------

	runCmd := fmt.Sprintf("chmod +x %s && %s apply --config -", remoteBinPath, remoteBinPath)
	if e.Opts.DryRun {
		runCmd += " --dry-run"
	}

	// Konfigürasyonu stdin üzerinden gönder
	err = t.RunRemoteSecure(ctx, runCmd, target.BecomePassword, string(configContent))

	if err != nil {
		return 0, fmt.Errorf("uzak sunucu hatası: %w", err)
	}

	if !e.Opts.DryRun {
		localTempState := fmt.Sprintf("/tmp/monarch-state-%s.json", timestamp)
		remoteStatePath := ".monarch/state.json"

		if downloadErr := t.DownloadFile(ctx, remoteStatePath, localTempState); downloadErr == nil {
			fileData, readErr := os.ReadFile(localTempState)
			if readErr == nil {
				var remoteState State
				if jsonErr := json.Unmarshal(fileData, &remoteState); jsonErr == nil {
					e.stateMutex.Lock()
					e.State.Merge(&remoteState)
					_ = e.State.Save()
					e.stateMutex.Unlock()
					slog.Info("Uzak state senkronize edildi.")
				}
			}
			_ = os.Remove(localTempState)
		}
	}

	return 0, nil
}

// resolveBinaryPath: Hedef mimari için gömülü binary'i geçici dizine çıkarır.
func resolveBinaryPath(targetOS, targetArch string) (string, error) {
	// 1. Eğer hedef mimari, çalışan makineyle aynıysa ve dev modundaysak (embed yoksa)
	// Kendi executable dosyamızı kullanabiliriz. Ancak prodüksiyonda embed tercih edilir.
	// Tutarlılık için önce embed kontrol edelim.

	binaryName := fmt.Sprintf("monarch-%s-%s", targetOS, targetArch)
	embedPath := fmt.Sprintf("embedded/%s", binaryName)

	// 2. Gömülü dosya var mı kontrol et
	content, err := embeddedBinaries.ReadFile(embedPath)
	if err != nil {
		// Embed içinde yoksa ve localde çalışıyorsak (geliştirme ortamı), belki henüz 'make' çalıştırılmamıştır.
		// Son çare olarak çalışan binary'yi (kendimizi) önerelim mi?
		// Sadece OS/Arch tutuyorsa.
		if targetOS == runtime.GOOS && targetArch == runtime.GOARCH {
			slog.Warn("Gömülü binary bulunamadı, mevcut çalışan dosya kullanılıyor.", "missing", binaryName)
			return os.Executable()
		}

		return "", fmt.Errorf("bu mimari için gömülü binary bulunamadı (%s). Lütfen 'make' komutu ile derleme yapın", binaryName)
	}

	// 3. Gömülü dosyayı geçici bir yere çıkar (Extract)
	tempFile, err := os.CreateTemp("", binaryName+"-*")
	if err != nil {
		return "", fmt.Errorf("geçici dosya oluşturulamadı: %w", err)
	}
	defer tempFile.Close()

	if _, err := tempFile.Write(content); err != nil {
		return "", fmt.Errorf("binary diske yazılamadı: %w", err)
	}

	// Çalıştırılabilir yap
	if err := tempFile.Chmod(0755); err != nil {
		return "", fmt.Errorf("chmod hatası: %w", err)
	}

	slog.Info("Gömülü binary çıkarıldı", "path", tempFile.Name())
	return tempFile.Name(), nil
}
