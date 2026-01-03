package docker

import (
	"encoding/json"
	"fmt"

	"github.com/melih-ucgun/veto/internal/core"
)

func init() {
	core.RegisterResource("docker_container", NewContainerAdapter)
}

type ContainerAdapter struct {
	Name   string
	State  string // running, stopped, absent
	Params map[string]interface{}
}

func NewContainerAdapter(name string, params map[string]interface{}, ctx *core.SystemContext) (core.Resource, error) {
	// The core engine Mapping:
	// ResourceConfig has "State".
	// But Adapter factory takes "params".
	// Usually Adapter handles its own state logic if passed in Params OR core passes it?
	// Wait, factory signature is `New(name, params, ctx)`.
	// Core engine handles `State` field in `ResourceConfig`, but doesn't pass it explicitly to New?
	// Let's check `factory.go` or `engine.go`.
	// `engine.go` calls `res, err := factory.New(it.Type, it.Name, it.Params, e.Context)`
	// `it` (ResourceConfig) has `State`.
	// BUT `it.Params` does NOT automatically include State unless user put it there.
	// My design in `implementation_plan` had `state: running` at top level.
	// The `params` map only has what's under `params:`.
	// So `State` is NOT in `params`.
	// This is a known issue/design choice in Veto.
	// Most adapters (file, pkg) expect `state` in `params` or implied?
	// Let's check `file` adapter.
	// Oh, `file` adapter usually uses `ensure: present/absent` in params or similar?
	// Or `State` field in implementation plan was just illustrative or I need to put it in Params?
	// In `Apply`, I should ideally respect `it.State`.
	// But `ExecuteLayer` calls `res.Apply`.
	// The ADAPTER needs to know the desired state to Check against.
	// `Check` returns `(drifted bool, err)`.
	// If `Check` doesn't know desired state, it can't know if drifted.
	// So Desired State MUST be passed to Adapter.
	// Current `factory` signature `New(name, params, ctx)` implies `State` must be in `params`.
	// USER must put `state: running` inside `params`.
	// OR `Config` loader merges top-level `state` into params?
	// Let's assume user puts it in `params` OR I default to "running".
	// The `docker_container` schema in plan shows `state` at TOP LEVEL.
	// This means `ExecuteLayer` logic:
	// If `Adapter` implements some `SetState` interface? No.
	// I should probably add `state` to params in `config` expansion?
	// OR Just tell user to put it in params.
	// Let's handle it by reading from params. Implementation Plan example:
	// params:
	//   image: ...
	//   state: running  <-- I'll support this.
	// If top level `state` is used, it's currently ignored by Adapter.
	// I'll stick to `params["state"]`. Default to "running".

	desiredState, _ := params["state"].(string)
	if desiredState == "" {
		desiredState = "running"
	}

	return &ContainerAdapter{
		Name:   name,
		State:  desiredState,
		Params: params,
	}, nil
}

func (a *ContainerAdapter) GetName() string { return a.Name }
func (a *ContainerAdapter) GetType() string { return "docker_container" }

func (a *ContainerAdapter) Validate(ctx *core.SystemContext) error {
	if a.Params["image"] == "" && a.State != "absent" {
		return fmt.Errorf("image is required for container %s", a.Name)
	}
	return nil
}

// DockerInspect subset
type InspectResult struct {
	State struct {
		Running bool
		Status  string
	}
	Config struct {
		Image string
	}
	Image string // ID
}

func (a *ContainerAdapter) Check(ctx *core.SystemContext) (bool, error) {
	// 1. Inspect
	out, err := ctx.Transport.Execute(ctx.Context, fmt.Sprintf("docker inspect %s", a.Name))

	exists := err == nil

	if a.State == "absent" {
		return exists, nil // If exists, we need change (remove).
	}

	if !exists {
		return true, nil // Need to create
	}

	// Parse JSON to check status
	var results []InspectResult
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		return true, fmt.Errorf("failed to parse docker inspect: %w", err) // Assume drift if parse fails
	}
	if len(results) == 0 {
		return true, nil // Weird
	}
	container := results[0]

	// Check State
	if a.State == "running" && !container.State.Running {
		return true, nil // Need start
	}
	if a.State == "stopped" && container.State.Running {
		return true, nil // Need stop
	}

	// Check Image Drift (Simple check against config image name)
	// Note: `docker run ubuntu` -> Config.Image = "ubuntu".
	// But `docker inspect` might return full hash if inspected differently or verified?
	// Usually Config.Image preserves what was passed.
	// If user says "nginx:latest", we check if container was created with "nginx:latest".
	desiredImage, _ := a.Params["image"].(string)
	if desiredImage != "" && container.Config.Image != desiredImage {
		// Image changed?
		// Note: if user used "nginx" and docker resolved to "nginx:latest", string match fails.
		// We'll require exact match for now.
		return true, nil
	}

	return false, nil
}

func (a *ContainerAdapter) Apply(ctx *core.SystemContext) (core.Result, error) {
	// Re-check existence to decide action (Create vs Start vs Stop vs Remove)
	// Or we can rely on `Check` logic but `Apply` needs to be robust.

	// Helper to run
	run := func(cmd string) (string, error) {
		return ctx.Transport.Execute(ctx.Context, cmd)
	}

	out, err := run(fmt.Sprintf("docker inspect %s", a.Name))
	exists := err == nil
	var container InspectResult
	if exists {
		var results []InspectResult
		_ = json.Unmarshal([]byte(out), &results)
		if len(results) > 0 {
			container = results[0]
		}
	}

	if a.State == "absent" {
		if exists {
			if _, err := run(fmt.Sprintf("docker rm -f %s", a.Name)); err != nil {
				return core.Failure(err, "Failed to remove container"), err
			}
			return core.SuccessChange("Container removed"), nil
		}
		return core.SuccessNoChange("Container already absent"), nil
	}

	desiredImage, _ := a.Params["image"].(string)

	// If exists
	if exists {
		// Check drift (Image)
		imageChanged := desiredImage != "" && container.Config.Image != desiredImage

		if imageChanged {
			// Recreate
			if _, err := run(fmt.Sprintf("docker rm -f %s", a.Name)); err != nil {
				return core.Failure(err, "Failed to remove for recreation"), err
			}
			exists = false // Proceed to create
		} else {
			// Update state
			if a.State == "running" && !container.State.Running {
				if _, err := run(fmt.Sprintf("docker start %s", a.Name)); err != nil {
					return core.Failure(err, "Failed to start container"), err
				}
				return core.SuccessChange("Container started"), nil
			}
			if a.State == "stopped" && container.State.Running {
				if _, err := run(fmt.Sprintf("docker stop %s", a.Name)); err != nil {
					return core.Failure(err, "Failed to stop container"), err
				}
				return core.SuccessChange("Container stopped"), nil
			}
			return core.SuccessNoChange("Container already in desired state"), nil
		}
	}

	// Create (Run)
	if a.State == "running" || a.State == "stopped" { // "stopped" creation implies create but don't start? `docker create`?
		// For simplicity, "stopped" -> `docker create`?
		// Most users use `running`.
		// If `stopped`, we verify existence above. If not exists, we create it?
		// Let's support `docker run -d` for running.
		// If stopped requested, we run then stop? Or `docker create`.
		// Let's use `docker run -d` if running.

		if a.State == "stopped" {
			// Advanced: docker create.
			// MVP: Fail or warn? Or just create and stop.
			// Let's focus on `running`.
		}

		cmd := fmt.Sprintf("docker run -d --name %s", a.Name)

		// Map ports
		if ports, ok := a.Params["ports"].([]interface{}); ok {
			for _, p := range ports {
				cmd += fmt.Sprintf(" -p %v", p)
			}
		}

		// Map volumes
		if vols, ok := a.Params["volumes"].([]interface{}); ok {
			for _, v := range vols {
				cmd += fmt.Sprintf(" -v %v", v)
			}
		}

		// Env
		if env, ok := a.Params["env"].(map[string]interface{}); ok {
			for k, v := range env {
				cmd += fmt.Sprintf(" -e %s='%v'", k, v)
			}
		}

		// Restart policy
		if restart, ok := a.Params["restart"].(string); ok {
			cmd += fmt.Sprintf(" --restart %s", restart)
		}

		cmd += " " + desiredImage

		if _, err := run(cmd); err != nil {
			return core.Failure(err, "Failed to run container"), err
		}

		// If stopped was requested (weird for new container but possible)
		if a.State == "stopped" {
			run(fmt.Sprintf("docker stop %s", a.Name))
			return core.SuccessChange("Container created (stopped)"), nil
		}

		return core.SuccessChange("Container started"), nil
	}

	return core.SuccessNoChange("No action needed"), nil
}
