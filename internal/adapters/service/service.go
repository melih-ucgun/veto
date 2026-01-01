package service

import (
	"fmt"
	"strings"

	"github.com/melih-ucgun/veto/internal/core"
)

type ServiceAdapter struct {
	core.BaseResource
	State           string // active, stopped, restarted
	Enabled         bool   // true, false
	Manager         ServiceManager
	ActionPerformed []string // To track actions for Revert (e.g., "started", "enabled")
}

func NewServiceAdapter(name string, params map[string]interface{}, ctx *core.SystemContext) *ServiceAdapter {
	state, _ := params["state"].(string)
	if state == "" {
		state = "active"
	}

	enabled := true
	if e, ok := params["enabled"].(bool); ok {
		enabled = e
	}

	mgr := GetServiceManager(ctx)

	return &ServiceAdapter{
		BaseResource: core.BaseResource{Name: name, Type: "service"},
		State:        state,
		Enabled:      enabled,
		Manager:      mgr,
	}
}

func (r *ServiceAdapter) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("service name is required")
	}
	return nil
}

func (r *ServiceAdapter) Check(ctx *core.SystemContext) (bool, error) {
	// 1. Enable Check
	isEnabled, err := r.Manager.IsEnabled(r.Name)
	if err != nil {
		return false, err
	}
	if r.Enabled != isEnabled {
		return true, nil
	}

	// 2. Active Check
	isActive, err := r.Manager.IsActive(r.Name)
	if err != nil {
		return false, err
	}

	// Restart her zaman eylem gerektirir (check'i true döndürür) ancak "restarted" dry-run'da sürekli çalışır.
	// İdeal state "started" olmalı. "restarted" bir eylemdir.
	// Eğer state active ise ve çalışıyorsa bir şey yapma.

	if r.State == "restarted" {
		return true, nil
	}

	if r.State == "stopped" {
		return isActive, nil // Çalışıyorsa durdurulmalı -> true
	}

	// state == active/started
	return !isActive, nil // Çalışmıyorsa başlatılmalı -> true
}

func (r *ServiceAdapter) Apply(ctx *core.SystemContext) (core.Result, error) {
	needsAction, err := r.Check(ctx)
	if err != nil {
		return core.Failure(err, "Check failed"), err
	}
	if !needsAction {
		return core.SuccessNoChange(fmt.Sprintf("Service %s is in desired state", r.Name)), nil
	}

	if ctx.DryRun {
		return core.SuccessChange(fmt.Sprintf("[DryRun] Configure service %s (enable=%v, state=%s) via %s", r.Name, r.Enabled, r.State, r.Manager.Name())), nil
	}

	messages := []string{}
	r.ActionPerformed = []string{}

	// 1. Enable/Disable
	// Enable logic checks current state first to be safe, or just forces it.
	// Check method already verified current != desired. but Apply might run without Check returning specifics.
	// But let's just do it. Idempotency is handled by systemd mostly.

	currentEnabled, _ := r.Manager.IsEnabled(r.Name)
	if currentEnabled != r.Enabled {
		if r.Enabled {
			if err := r.Manager.Enable(r.Name); err != nil {
				return core.Failure(err, "Failed to enable service"), err
			}
			r.ActionPerformed = append(r.ActionPerformed, "enabled")
			messages = append(messages, "Service enabled")
		} else {
			if err := r.Manager.Disable(r.Name); err != nil {
				return core.Failure(err, "Failed to disable service"), err
			}
			r.ActionPerformed = append(r.ActionPerformed, "disabled")
			messages = append(messages, "Service disabled")
		}
	}

	// 2. State Change
	currentActive, _ := r.Manager.IsActive(r.Name)

	if r.State == "active" || r.State == "started" {
		if !currentActive {
			if err := r.Manager.Start(r.Name); err != nil {
				return core.Failure(err, "Failed to start service"), err
			}
			r.ActionPerformed = append(r.ActionPerformed, "started")
			messages = append(messages, "Service started")
		}
	} else if r.State == "stopped" {
		if currentActive {
			if err := r.Manager.Stop(r.Name); err != nil {
				return core.Failure(err, "Failed to stop service"), err
			}
			r.ActionPerformed = append(r.ActionPerformed, "stopped")
			messages = append(messages, "Service stopped")
		}
	} else if r.State == "restarted" {
		if err := r.Manager.Restart(r.Name); err != nil {
			return core.Failure(err, "Failed to restart service"), err
		}
		r.ActionPerformed = append(r.ActionPerformed, "restarted")
		messages = append(messages, "Service restarted")
	}

	return core.SuccessChange(strings.Join(messages, ", ")), nil
}

func (r *ServiceAdapter) Revert(ctx *core.SystemContext) error {
	// Revert order: reverse of apply
	for i := len(r.ActionPerformed) - 1; i >= 0; i-- {
		action := r.ActionPerformed[i]
		switch action {
		case "started":
			// Stop it
			r.Manager.Stop(r.Name)
		case "stopped":
			// Start it (might be risky if it was broken before, but revert means undo)
			r.Manager.Start(r.Name)
		case "restarted":
			// Can't really "un-restart". Maybe restart again? Or ignore.
			// Ignoring is safer.
		case "enabled":
			// Disable it
			r.Manager.Disable(r.Name)
		case "disabled":
			// Enable it
			r.Manager.Enable(r.Name)
		}
	}
	return nil
}

// ListInstalled satisfies the core.Lister interface for Prune operations.
// It returns a list of enabled services.
func (r *ServiceAdapter) ListInstalled(ctx *core.SystemContext) ([]string, error) {
	services, err := r.Manager.ListEnabled()
	if err != nil {
		return nil, fmt.Errorf("failed to list enabled services: %w", err)
	}
	return services, nil
}
