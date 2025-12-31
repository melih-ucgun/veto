package identity

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/melih-ucgun/monarch/internal/core"
)

type UserAdapter struct {
	core.BaseResource
	Uid             string
	Gid             string
	Groups          []string // Ek gruplar
	Home            string
	Shell           string
	System          bool
	State           string
	ActionPerformed string
}

func NewUserAdapter(name string, params map[string]interface{}) *UserAdapter {
	uid, _ := params["uid"].(string)
	gid, _ := params["gid"].(string)
	home, _ := params["home"].(string)
	shell, _ := params["shell"].(string)

	system := false
	if s, ok := params["system"].(bool); ok {
		system = s
	}

	var groups []string
	if gList, ok := params["groups"].([]interface{}); ok {
		for _, g := range gList {
			if gStr, ok := g.(string); ok {
				groups = append(groups, gStr)
			}
		}
	} else if gStr, ok := params["groups"].(string); ok {
		// Virgülle ayrılmış string desteği
		groups = strings.Split(gStr, ",")
	}

	state, _ := params["state"].(string)
	if state == "" {
		state = "present"
	}

	return &UserAdapter{
		BaseResource: core.BaseResource{Name: name, Type: "user"},
		Uid:          uid,
		Gid:          gid,
		Groups:       groups,
		Home:         home,
		Shell:        shell,
		System:       system,
		State:        state,
	}
}

func (r *UserAdapter) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("username is required")
	}
	return nil
}

func (r *UserAdapter) Check(ctx *core.SystemContext) (bool, error) {
	// id -u <user>
	cmd := exec.Command("id", "-u", r.Name)
	err := cmd.Run()
	exists := (err == nil)

	if r.State == "absent" {
		return exists, nil
	}
	if !exists {
		return true, nil
	}

	// Kullanıcı var, özelliklerini kontrol et (Shell, Groups vb.)
	// Detaylı kontrol şimdilik atlanıyor, production'da 'id <user>' çıktısı parse edilmeli.
	// Basitlik için sadece varlık kontrolü yapıyoruz.
	return false, nil
}

func (r *UserAdapter) Apply(ctx *core.SystemContext) (core.Result, error) {
	needsAction, _ := r.Check(ctx)
	if !needsAction {
		return core.SuccessNoChange(fmt.Sprintf("User %s is correct", r.Name)), nil
	}

	if ctx.DryRun {
		return core.SuccessChange(fmt.Sprintf("[DryRun] User %s state=%s", r.Name, r.State)), nil
	}

	if r.State == "absent" {
		if out, err := exec.Command("userdel", "-r", r.Name).CombinedOutput(); err != nil {
			return core.Failure(err, "Failed to delete user: "+string(out)), err
		}
		r.ActionPerformed = "deleted"
		return core.SuccessChange("User deleted"), nil
	}

	// Kullanıcı oluştur
	args := []string{}
	if r.Uid != "" {
		args = append(args, "-u", r.Uid)
	}
	if r.Gid != "" {
		args = append(args, "-g", r.Gid)
	}
	if r.Home != "" {
		args = append(args, "-d", r.Home, "-m")
	}
	if r.Shell != "" {
		args = append(args, "-s", r.Shell)
	}
	if len(r.Groups) > 0 {
		args = append(args, "-G", strings.Join(r.Groups, ","))
	}
	if r.System {
		args = append(args, "-r")
	}

	args = append(args, r.Name)

	if out, err := exec.Command("useradd", args...).CombinedOutput(); err != nil {
		return core.Failure(err, "Failed to create user: "+string(out)), err
	}
	r.ActionPerformed = "created"

	return core.SuccessChange("User created"), nil
}

func (r *UserAdapter) Revert(ctx *core.SystemContext) error {
	if r.ActionPerformed == "created" {
		// Oluşturulan kullanıcıyı sil
		if out, err := exec.Command("userdel", "-r", r.Name).CombinedOutput(); err != nil {
			return fmt.Errorf("failed to revert user creation: %s: %w", out, err)
		}
	}
	return nil
}
