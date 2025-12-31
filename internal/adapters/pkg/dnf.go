package pkg

import (
	"fmt"

	"github.com/melih-ucgun/monarch/internal/core"
)

type DnfAdapter struct {
	core.BaseResource
	State           string
	ActionPerformed string
}

func NewDnfAdapter(name string, state string) *DnfAdapter {
	if state == "" {
		state = "present"
	}
	return &DnfAdapter{
		BaseResource: core.BaseResource{Name: name, Type: "package"},
		State:        state,
	}
}

func (r *DnfAdapter) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("package name is required for dnf")
	}
	return nil
}

func (r *DnfAdapter) Check(ctx *core.SystemContext) (bool, error) {
	installed := isInstalled("rpm", "-q", r.Name)
	if r.State == "absent" {
		return installed, nil
	}
	return !installed, nil
}

func (r *DnfAdapter) Apply(ctx *core.SystemContext) (core.Result, error) {
	needsAction, _ := r.Check(ctx)
	if !needsAction {
		return core.SuccessNoChange(fmt.Sprintf("Package %s already %s", r.Name, r.State)), nil
	}

	if ctx.DryRun {
		return core.SuccessChange(fmt.Sprintf("[DryRun] dnf %s %s", r.State, r.Name)), nil
	}

	var args []string
	if r.State == "absent" {
		args = []string{"remove", "-y", r.Name}
		r.ActionPerformed = "removed"
	} else {
		args = []string{"install", "-y", r.Name}
		r.ActionPerformed = "installed"
	}

	out, err := runCommand("dnf", args...)
	if err != nil {
		r.ActionPerformed = ""
		return core.Failure(err, "Dnf failed: "+out), err
	}

	return core.SuccessChange(fmt.Sprintf("Dnf processed %s", r.Name)), nil
}

func (r *DnfAdapter) Revert(ctx *core.SystemContext) error {
	if r.ActionPerformed == "installed" {
		_, err := runCommand("dnf", "remove", "-y", r.Name)
		return err
	} else if r.ActionPerformed == "removed" {
		_, err := runCommand("dnf", "install", "-y", r.Name)
		return err
	}
	return nil
}
