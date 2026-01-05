package identity

import (
	"fmt"
	"strings"

	"github.com/melih-ucgun/veto/internal/core"
)

func init() {
	core.RegisterResource("user", func(name string, params map[string]interface{}, ctx *core.SystemContext) (core.Resource, error) {
		return NewUserAdapter(name, params), nil
	})
}

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

func NewUserAdapter(name string, params map[string]interface{}) core.Resource {
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

func (r *UserAdapter) Validate(ctx *core.SystemContext) error {
	if r.Name == "" {
		return fmt.Errorf("username is required")
	}
	return nil
}

// Check verifies if the user exists and matches the desired state.
func (r *UserAdapter) Check(ctx *core.SystemContext) (bool, error) {
	// getent passwd <user>
	// Output format: username:password:uid:gid:gecos:home:shell
	out, err := ctx.Transport.Execute(ctx.Context, "getent passwd "+r.Name)
	exists := (err == nil)

	if r.State == "absent" {
		return exists, nil
	}

	if !exists {
		return true, nil // Needs to be created
	}

	// Parse output
	parts := strings.Split(strings.TrimSpace(out), ":")
	if len(parts) < 7 {
		return false, fmt.Errorf("unexpected output from getent: %s", out)
	}

	currentUid := parts[2]
	currentGid := parts[3]
	currentHome := parts[5]
	currentShell := parts[6]

	// Check UID
	if r.Uid != "" && r.Uid != currentUid {
		return true, nil
	}

	// Check GID
	if r.Gid != "" && r.Gid != currentGid {
		return true, nil
	}

	// Check Home
	if r.Home != "" && r.Home != currentHome {
		return true, nil
	}

	// Check Shell
	if r.Shell != "" && r.Shell != currentShell {
		return true, nil
	}

	// Check Groups
	// id -Gn <user> -> returns space separated list of groups
	if len(r.Groups) > 0 {
		outGroups, err := ctx.Transport.Execute(ctx.Context, "id -Gn "+r.Name)
		if err != nil {
			return false, fmt.Errorf("failed to get user groups: %w", err)
		}
		currentGroups := strings.Fields(outGroups)

		// Create a map for easy lookup
		groupMap := make(map[string]bool)
		for _, g := range currentGroups {
			groupMap[g] = true
		}

		// Verify all desired groups are present
		// Note: This logic ensures desired groups are present (append behavior).
		// If we want strict equality (remove extra groups), logic differs.
		// Standard ansible 'groups' param usually implies "set these groups" (replace)
		// if append=no (default), or add if append=yes.
		// We'll assume declarative "these should be the groups".
		// For now let's just check if missing any.
		for _, g := range r.Groups {
			if !groupMap[g] {
				return true, nil
			}
		}

		// If we want strict equality (no extra groups), we should check counts too?
		// "id -Gn" result includes primary group. r.Groups usually lists secondary groups.
		// Let's stick to "ensure these groups exist" for now to avoid complexity with primary group.
	}

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
		fullCmd := "userdel -r " + r.Name
		if out, err := ctx.Transport.Execute(ctx.Context, fullCmd); err != nil {
			return core.Failure(err, "Failed to delete user: "+out), err
		}
		r.ActionPerformed = "deleted"
		return core.SuccessChange("User deleted"), nil
	}

	// Check if user exists to decide between useradd (create) and usermod (modify)
	exists := false
	if _, err := ctx.Transport.Execute(ctx.Context, "id -u "+r.Name); err == nil {
		exists = true
	}

	cmd := "useradd"
	if exists {
		cmd = "usermod"
	}

	args := []string{}
	if r.Uid != "" {
		args = append(args, "-u", r.Uid)
	}
	if r.Gid != "" {
		args = append(args, "-g", r.Gid)
	}
	if r.Home != "" {
		if exists {
			args = append(args, "-d", r.Home, "-m") // Move content if modifying
		} else {
			args = append(args, "-d", r.Home, "-m")
		}
	}
	if r.Shell != "" {
		args = append(args, "-s", r.Shell)
	}
	if len(r.Groups) > 0 {
		// -G sets groups (replace secondary). -aG appends.
		// Let's use -G for declarative (exact list of secondary groups).
		// Note: useradd/usermod -G takes comma separated list
		args = append(args, "-G", strings.Join(r.Groups, ","))
	}

	if !exists && r.System {
		args = append(args, "-r")
	}

	args = append(args, r.Name)

	fullCmd := cmd + " " + strings.Join(args, " ")
	if out, err := ctx.Transport.Execute(ctx.Context, fullCmd); err != nil {
		return core.Failure(err, fmt.Sprintf("Failed to %s user: %s", cmd, out)), err
	}

	if exists {
		r.ActionPerformed = "modified"
		return core.SuccessChange("User modified"), nil
	}

	r.ActionPerformed = "created"
	return core.SuccessChange("User created"), nil
}

func (r *UserAdapter) Revert(ctx *core.SystemContext) error {
	if r.ActionPerformed == "created" {
		// Oluşturulan kullanıcıyı sil
		fullCmd := "userdel -r " + r.Name
		if out, err := ctx.Transport.Execute(ctx.Context, fullCmd); err != nil {
			return fmt.Errorf("failed to revert user creation: %s: %w", out, err)
		}
	}
	// Cannot easily revert modification without storing previous state
	return nil
}
