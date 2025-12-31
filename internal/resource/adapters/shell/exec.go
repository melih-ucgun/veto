package shell

import (
	"fmt"
	"os/exec"

	"github.com/melih-ucgun/monarch/internal/core"
)

type ExecAdapter struct {
	core.BaseResource
	Command       string
	Unless        string // Eğer bu komut başarılı olursa (exit 0), ana komutu çalıştırma
	OnlyIf        string // Sadece bu komut başarılı olursa ana komutu çalıştır
	RevertCommand string // Rollback durumunda çalıştırılacak komut
}

func NewExecAdapter(name string, params map[string]interface{}) *ExecAdapter {
	cmd, _ := params["command"].(string)
	if cmd == "" {
		cmd = name
	} // Command verilmezse isim komut olarak kullanılır

	unless, _ := params["unless"].(string)
	onlyif, _ := params["onlyif"].(string)
	revertCmd, _ := params["revert_command"].(string)

	return &ExecAdapter{
		BaseResource:  core.BaseResource{Name: name, Type: "exec"},
		Command:       cmd,
		Unless:        unless,
		OnlyIf:        onlyif,
		RevertCommand: revertCmd,
	}
}

func (r *ExecAdapter) Validate() error {
	if r.Command == "" {
		return fmt.Errorf("command is required for exec resource")
	}
	return nil
}

func (r *ExecAdapter) Check(ctx *core.SystemContext) (bool, error) {
	// 1. Unless Kontrolü: Eğer başarılı olursa, çalışmaya GEREK YOK (return false)
	if r.Unless != "" {
		err := exec.Command("sh", "-c", r.Unless).Run()
		if err == nil {
			return false, nil // Unless başarılı, işlem yapma
		}
	}

	// 2. OnlyIf Kontrolü: Eğer başarısız olursa, çalışmaya GEREK YOK (return false)
	if r.OnlyIf != "" {
		err := exec.Command("sh", "-c", r.OnlyIf).Run()
		if err != nil {
			return false, nil // OnlyIf başarısız, işlem yapma
		}
	}

	// Exec her zaman "değişiklik yapacak" kabul edilir (unless/onlyif yoksa)
	return true, nil
}

func (r *ExecAdapter) Apply(ctx *core.SystemContext) (core.Result, error) {
	shouldRun, _ := r.Check(ctx)
	if !shouldRun {
		return core.SuccessNoChange("Skipped due to unless/onlyif conditions"), nil
	}

	if ctx.DryRun {
		return core.SuccessChange(fmt.Sprintf("[DryRun] Exec: %s", r.Command)), nil
	}

	out, err := exec.Command("sh", "-c", r.Command).CombinedOutput()
	if err != nil {
		return core.Failure(err, fmt.Sprintf("Command failed: %s", string(out))), err
	}

	return core.SuccessChange("Command executed successfully"), nil
}

func (r *ExecAdapter) Revert(ctx *core.SystemContext) error {
	if r.RevertCommand != "" {
		out, err := exec.Command("sh", "-c", r.RevertCommand).CombinedOutput()
		if err != nil {
			return fmt.Errorf("revert command failed: %s: %w", out, err)
		}
	}
	return nil
}
