package identity

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/melih-ucgun/veto/internal/core"
)

func init() {
	core.RegisterResource("group", func(name string, params map[string]interface{}, ctx *core.SystemContext) (core.Resource, error) {
		return NewGroupAdapter(name, params), nil
	})
}

type GroupAdapter struct {
	core.BaseResource
	Gid             int
	System          bool
	State           string
	ActionPerformed string
}

func NewGroupAdapter(name string, params map[string]interface{}) core.Resource {
	gid := -1
	if g, ok := params["gid"].(int); ok {
		gid = g
	} else if gStr, ok := params["gid"].(string); ok {
		gid, _ = strconv.Atoi(gStr)
	}

	system := false
	if s, ok := params["system"].(bool); ok {
		system = s
	}

	state, _ := params["state"].(string)
	if state == "" {
		state = "present"
	}

	return &GroupAdapter{
		BaseResource: core.BaseResource{Name: name, Type: "group"},
		Gid:          gid,
		System:       system,
		State:        state,
	}
}

func (r *GroupAdapter) Validate(ctx *core.SystemContext) error {
	if r.Name == "" {
		return fmt.Errorf("group name is required")
	}
	return nil
}

func (r *GroupAdapter) Check(ctx *core.SystemContext) (bool, error) {
	// getent group <name>
	_, err := ctx.Transport.Execute(ctx.Context, "getent group "+r.Name)
	exists := (err == nil)

	if r.State == "absent" {
		return exists, nil
	}

	if !exists {
		return true, nil // Grup yok, oluşturulmalı
	}

	// Grup var, GID kontrolü (opsiyonel)
	if r.Gid != -1 {
		// getent çıktısını parse et: root:x:0:
		out, _ := ctx.Transport.Execute(ctx.Context, "getent group "+r.Name)
		parts := strings.Split(strings.TrimSpace(out), ":")
		if len(parts) >= 3 {
			currentGid, _ := strconv.Atoi(parts[2])
			if currentGid != r.Gid {
				return true, nil // GID değişmeli
			}
		}
	}

	return false, nil
}

func (r *GroupAdapter) Apply(ctx *core.SystemContext) (core.Result, error) {
	needsAction, _ := r.Check(ctx)
	if !needsAction {
		return core.SuccessNoChange(fmt.Sprintf("Group %s is correct", r.Name)), nil
	}

	if ctx.DryRun {
		return core.SuccessChange(fmt.Sprintf("[DryRun] Group %s state=%s", r.Name, r.State)), nil
	}

	if r.State == "absent" {
		fullCmd := "groupdel " + r.Name
		if out, err := ctx.Transport.Execute(ctx.Context, fullCmd); err != nil {
			return core.Failure(err, "Failed to delete group: "+out), err
		}
		r.ActionPerformed = "deleted"
		return core.SuccessChange("Group deleted"), nil
	}

	// Grup oluşturma veya güncelleme
	// Basitlik adına sadece oluşturmayı ele alalım, modifikasyon için groupmod kullanılabilir.
	// Eğer grup zaten varsa ama GID yanlışsa, groupmod çalıştırılmalı.

	if _, err := ctx.Transport.Execute(ctx.Context, "getent group "+r.Name); err == nil {
		// Grup var, güncelle (groupmod)
		args := []string{}
		if r.Gid != -1 {
			args = append(args, "-g", strconv.Itoa(r.Gid))
		}
		args = append(args, r.Name)

		if len(args) > 1 { // Sadece isim değil, argüman da varsa
			fullCmd := "groupmod " + strings.Join(args, " ")
			if out, err := ctx.Transport.Execute(ctx.Context, fullCmd); err != nil {
				return core.Failure(err, "Failed to modify group: "+out), err
			}
			r.ActionPerformed = "modified"
			return core.SuccessChange("Group modified"), nil
		}
		return core.SuccessNoChange("Group exists"), nil
	}

	// Yeni grup oluştur (groupadd)
	args := []string{}
	if r.Gid != -1 {
		args = append(args, "-g", strconv.Itoa(r.Gid))
	}
	if r.System {
		args = append(args, "-r")
	}
	args = append(args, r.Name)

	fullCmd := "groupadd " + strings.Join(args, " ")
	if out, err := ctx.Transport.Execute(ctx.Context, fullCmd); err != nil {
		return core.Failure(err, "Failed to create group: "+out), err
	}
	r.ActionPerformed = "created"

	return core.SuccessChange("Group created"), nil
}

func (r *GroupAdapter) Revert(ctx *core.SystemContext) error {
	if r.ActionPerformed == "created" {
		fullCmd := "groupdel " + r.Name
		if out, err := ctx.Transport.Execute(ctx.Context, fullCmd); err != nil {
			return fmt.Errorf("failed to revert group creation: %s: %w", out, err)
		}
	}
	return nil
}
