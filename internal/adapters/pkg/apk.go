package pkg

import (
	"fmt"

	"github.com/melih-ucgun/monarch/internal/core"
)

type ApkAdapter struct {
	core.BaseResource
	State string
}

func NewApkAdapter(name string, state string) *ApkAdapter {
	if state == "" {
		state = "present"
	}
	return &ApkAdapter{
		BaseResource: core.BaseResource{Name: name, Type: "package"},
		State:        state,
	}
}

func (r *ApkAdapter) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("package name is required for apk")
	}
	return nil
}

func (r *ApkAdapter) Check(ctx *core.SystemContext) (bool, error) {
	// apk info -e <package> : Paket kuruluysa 0 döner
	installed := isInstalled("apk", "info", "-e", r.Name)

	if r.State == "absent" {
		return installed, nil
	}
	return !installed, nil
}

func (r *ApkAdapter) Apply(ctx *core.SystemContext) (core.Result, error) {
	needsAction, _ := r.Check(ctx)
	if !needsAction {
		return core.SuccessNoChange(fmt.Sprintf("Package %s already %s", r.Name, r.State)), nil
	}

	if ctx.DryRun {
		return core.SuccessChange(fmt.Sprintf("[DryRun] apk %s %s", r.State, r.Name)), nil
	}

	var args []string
	if r.State == "absent" {
		args = []string{"del", r.Name}
	} else {
		args = []string{"add", r.Name}
	}

	out, err := runCommand("apk", args...)
	if err != nil {
		return core.Failure(err, "Apk failed: "+out), err
	}

	return core.SuccessChange(fmt.Sprintf("Apk processed %s", r.Name)), nil
}

func (r *ApkAdapter) Revert(ctx *core.SystemContext) error {
	// Revert işlemini yapmamız için, az önce yaptığımız işlemin tersini yapmalıyız.
	// Eğer State="present" ise -> "absent" yap (sil)
	// Eğer State="absent" ise -> "present" yap (kur)

	// TODO: Bu mantık "önceden sistemde var mıydı?" kontrolünü içermiyor.
	// İdeal dünyada, "önceden yoktu, ben kurdum, şimdi siliyorum" demeliyiz.
	// Ama basit atomic rollback için ters işlem şimdilik yeterli.

	var args []string
	if r.State == "present" {
		// Biz kurduk, geri alırken siliyoruz
		args = []string{"del", r.Name}
	} else {
		// Biz sildik, geri alırken kuruyoruz
		args = []string{"add", r.Name}
	}

	if ctx.DryRun {
		return nil
	}

	out, err := runCommand("apk", args...)
	if err != nil {
		return fmt.Errorf("revert apk failed: %s: %w", out, err)
	}
	return nil
}
