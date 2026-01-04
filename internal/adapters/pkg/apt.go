package pkg

import (
	"fmt"

	"github.com/melih-ucgun/veto/internal/core"
)

type AptAdapter struct {
	core.BaseResource
	State           string
	ActionPerformed string
}

func init() {
	core.RegisterResource("apt", func(name string, params map[string]interface{}, ctx *core.SystemContext) (core.Resource, error) {
		return NewAptAdapter(name, params), nil
	})
}

func NewAptAdapter(name string, params map[string]interface{}) core.Resource {
	state, _ := params["state"].(string)
	if state == "" {
		state = "present"
	}
	return &AptAdapter{
		BaseResource: core.BaseResource{Name: name, Type: "package"},
		State:        state,
	}
}

func (r *AptAdapter) Validate(ctx *core.SystemContext) error {
	if r.Name == "" {
		return fmt.Errorf("package name is required")
	}
	if r.State != "present" && r.State != "absent" {
		return fmt.Errorf("invalid state '%s', must be 'present' or 'absent'", r.State)
	}
	return nil
}

func (r *AptAdapter) Check(ctx *core.SystemContext) (bool, error) {
	// dpkg -s <package>
	installed := isInstalled(ctx, "dpkg", "-s", r.Name)

	if r.State == "absent" {
		return installed, nil
	}
	return !installed, nil
}

func (r *AptAdapter) Apply(ctx *core.SystemContext) (core.Result, error) {
	needsAction, _ := r.Check(ctx)
	if !needsAction {
		return core.SuccessNoChange(fmt.Sprintf("Package %s already %s", r.Name, r.State)), nil
	}

	if ctx.DryRun {
		return core.SuccessChange(fmt.Sprintf("[DryRun] apt-get %s %s", r.State, r.Name)), nil
	}

	var args []string
	if r.State == "absent" {
		args = []string{"remove", "-y", r.Name}
		r.ActionPerformed = "removed"
	} else {
		args = []string{"install", "-y", r.Name}
		r.ActionPerformed = "installed"
	}

	out, err := runCommand(ctx, "apt-get", args...)
	if err != nil {
		r.ActionPerformed = ""
		return core.Failure(err, "Apt failed: "+out), err
	}

	return core.SuccessChange(fmt.Sprintf("Apt processed %s", r.Name)), nil
}

func (r *AptAdapter) Revert(ctx *core.SystemContext) error {
	return r.RevertAction(r.ActionPerformed, ctx)
}

func (r *AptAdapter) RevertAction(action string, ctx *core.SystemContext) error {
	if action == "installed" {
		_, err := runCommand(ctx, "apt-get", "remove", "-y", r.Name)
		return err
	} else if action == "removed" {
		_, err := runCommand(ctx, "apt-get", "install", "-y", r.Name)
		return err
	}
	return nil
}
