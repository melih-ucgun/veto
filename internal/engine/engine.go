package engine

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/melih-ucgun/monarch/internal/config"
	"github.com/melih-ucgun/monarch/internal/resources"
	"github.com/melih-ucgun/monarch/internal/transport"
)

type EngineOptions struct {
	DryRun     bool
	AutoHeal   bool
	HostName   string
	ConfigFile string
}

type Reconciler struct {
	Config *config.Config
	Opts   EngineOptions
	State  *State
}

func NewReconciler(cfg *config.Config, opts EngineOptions) *Reconciler {
	state, _ := LoadState()
	return &Reconciler{Config: cfg, Opts: opts, State: state}
}

func (e *Reconciler) Run() (int, error) {
	if e.Opts.HostName == "" || e.Opts.HostName == "localhost" {
		return e.runLocal()
	}
	return e.runRemote()
}

func (e *Reconciler) runLocal() (int, error) {
	sorted, err := config.SortResources(e.Config.Resources)
	if err != nil {
		return 0, err
	}

	drifts := 0
	for _, rCfg := range sorted {
		res, err := resources.New(rCfg, e.Config.Vars)
		if err != nil || res == nil {
			slog.Warn("Kaynak oluşturulamadı", "name", rCfg.Name, "error", err)
			continue
		}

		ok, err := res.Check()
		if err != nil {
			slog.Error("Check hatası", "id", res.ID(), "err", err)
		}

		if !ok {
			drifts++
			diff, _ := res.Diff()
			if e.Opts.DryRun {
				slog.Info("SAPMA (Dry-Run)", "id", res.ID(), "diff", diff)
			} else {
				slog.Info("Uygulanıyor", "id", res.ID())
				applyErr := res.Apply()
				if applyErr != nil {
					slog.Error("Uygulama hatası", "id", res.ID(), "err", applyErr)
				}
				if e.State != nil {
					e.State.UpdateResource(res.ID(), rCfg.Type, applyErr == nil)
				}
			}
		}
	}
	if !e.Opts.DryRun && e.State != nil {
		_ = e.State.Save()
	}
	return drifts, nil
}

func (e *Reconciler) runRemote() (int, error) {
	var target *config.Host
	found := false
	for i := range e.Config.Hosts {
		if e.Config.Hosts[i].Name == e.Opts.HostName {
			target = &e.Config.Hosts[i]
			found = true
			break
		}
	}

	if !found {
		return 0, fmt.Errorf("host konfigürasyonda bulunamadı: %s", e.Opts.HostName)
	}

	// DÜZELTME: Hostname -> Host
	slog.Info("Uzak sunucuya bağlanılıyor...", "host", target.User)
	t, err := transport.NewSSHTransport(*target)
	if err != nil {
		return 0, fmt.Errorf("SSH bağlantı hatası: %v", err)
	}

	remoteOS, remoteArch, err := t.GetRemoteSystemInfo()
	if err != nil {
		return 0, fmt.Errorf("uzak sistem bilgisi alınamadı: %v", err)
	}
	slog.Info("Uzak sistem tespit edildi", "os", remoteOS, "arch", remoteArch)

	binaryPath, err := resolveBinaryPath(remoteOS, remoteArch)
	if err != nil {
		return 0, fmt.Errorf("binary hazırlama hatası: %v", err)
	}

	selfExe, _ := os.Executable()
	if binaryPath != selfExe {
		defer func() {
			slog.Debug("Geçici yerel binary siliniyor", "path", binaryPath)
			os.Remove(binaryPath)
		}()
	}

	timestamp := time.Now().Format("20060102150405")
	remoteBinPath := fmt.Sprintf("/tmp/monarch-%s", timestamp)
	remoteCfgPath := fmt.Sprintf("/tmp/monarch-%s.yaml", timestamp)

	slog.Info("Dosyalar gönderiliyor...", "bin", binaryPath, "config", e.Opts.ConfigFile)

	if err := t.CopyFile(binaryPath, remoteBinPath); err != nil {
		return 0, fmt.Errorf("binary kopyalama hatası: %v", err)
	}

	if err := t.CopyFile(e.Opts.ConfigFile, remoteCfgPath); err != nil {
		return 0, fmt.Errorf("config kopyalama hatası: %v", err)
	}

	runCmd := fmt.Sprintf("chmod +x %s && %s apply --config %s", remoteBinPath, remoteBinPath, remoteCfgPath)
	if target.BecomePassword != "" {
		runCmd = fmt.Sprintf("chmod +x %s && echo '%s' | sudo -S -p '' %s apply --config %s",
			remoteBinPath, target.BecomePassword, remoteBinPath, remoteCfgPath)
	}

	if e.Opts.DryRun {
		runCmd += " --dry-run"
	}

	slog.Info("Uzak komut çalıştırılıyor...")
	err = t.RunRemoteSecure(runCmd, "")

	cleanupCmd := fmt.Sprintf("rm -f %s %s", remoteBinPath, remoteCfgPath)
	if target.BecomePassword != "" {
		cleanupCmd = fmt.Sprintf("echo '%s' | sudo -S -p '' rm -f %s %s", target.BecomePassword, remoteBinPath, remoteCfgPath)
	}
	_ = t.RunRemoteSecure(cleanupCmd, "")

	return 0, err
}

func resolveBinaryPath(targetOS, targetArch string) (string, error) {
	localOS := runtime.GOOS
	localArch := runtime.GOARCH

	if targetOS == localOS && targetArch == localArch {
		exe, err := os.Executable()
		if err != nil {
			return "", err
		}
		slog.Info("Yerel mimari uyumlu, mevcut binary kullanılacak", "path", exe)
		return exe, nil
	}

	slog.Info("Mimari farklılığı tespit edildi, derleme deneniyor...",
		"local", fmt.Sprintf("%s/%s", localOS, localArch),
		"target", fmt.Sprintf("%s/%s", targetOS, targetArch))

	if _, err := exec.LookPath("go"); err != nil {
		return "", fmt.Errorf("'go' komutu bulunamadı. Çapraz derleme için Go yüklü olmalıdır")
	}

	if _, err := os.Stat("go.mod"); os.IsNotExist(err) {
		return "", fmt.Errorf("çapraz derleme için proje kök dizininde olmalısınız")
	}

	tempName := fmt.Sprintf("monarch-%s-%s", targetOS, targetArch)
	if targetOS == "windows" {
		tempName += ".exe"
	}
	tempPath := filepath.Join(os.TempDir(), tempName)

	cmd := exec.Command("go", "build", "-o", tempPath, ".")
	env := os.Environ()
	env = append(env, fmt.Sprintf("GOOS=%s", targetOS))
	env = append(env, fmt.Sprintf("GOARCH=%s", targetArch))
	env = append(env, "CGO_ENABLED=0")
	cmd.Env = env

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("derleme hatası:\n%s\nDetay: %v", string(output), err)
	}

	slog.Info("Çapraz derleme başarılı", "path", tempPath)
	return tempPath, nil
}
