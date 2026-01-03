package docker

import (
	"encoding/json"
	"fmt"

	"github.com/melih-ucgun/veto/internal/core"
)

func init() {
	core.RegisterResource("docker_container", NewDockerAdapter)
	core.RegisterResource("podman_container", NewPodmanAdapter)
}

type ContainerAdapter struct {
	Name   string
	State  string // running, stopped, absent
	Binary string // "docker" or "podman"
	Params map[string]interface{}
}

func NewDockerAdapter(name string, params map[string]interface{}, ctx *core.SystemContext) (core.Resource, error) {
	return newContainerAdapter(name, params, "docker")
}

func NewPodmanAdapter(name string, params map[string]interface{}, ctx *core.SystemContext) (core.Resource, error) {
	return newContainerAdapter(name, params, "podman")
}

func newContainerAdapter(name string, params map[string]interface{}, binary string) (core.Resource, error) {
	desiredState, _ := params["state"].(string)
	if desiredState == "" {
		desiredState = "running"
	}

	return &ContainerAdapter{
		Name:   name,
		State:  desiredState,
		Binary: binary,
		Params: params,
	}, nil
}

func (a *ContainerAdapter) GetName() string { return a.Name }
func (a *ContainerAdapter) GetType() string { return a.Binary + "_container" }

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
	out, err := ctx.Transport.Execute(ctx.Context, fmt.Sprintf("%s inspect %s", a.Binary, a.Name))

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
		return true, fmt.Errorf("failed to parse %s inspect: %w", a.Binary, err) // Assume drift if parse fails
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
	desiredImage, _ := a.Params["image"].(string)
	if desiredImage != "" && container.Config.Image != desiredImage {
		return true, nil
	}

	return false, nil
}

func (a *ContainerAdapter) Apply(ctx *core.SystemContext) (core.Result, error) {
	// Re-check logic
	run := func(cmd string) (string, error) {
		return ctx.Transport.Execute(ctx.Context, cmd)
	}

	out, err := run(fmt.Sprintf("%s inspect %s", a.Binary, a.Name))
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
			if _, err := run(fmt.Sprintf("%s rm -f %s", a.Binary, a.Name)); err != nil {
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
			if _, err := run(fmt.Sprintf("%s rm -f %s", a.Binary, a.Name)); err != nil {
				return core.Failure(err, "Failed to remove for recreation"), err
			}
			exists = false // Proceed to create
		} else {
			// Update state
			if a.State == "running" && !container.State.Running {
				if _, err := run(fmt.Sprintf("%s start %s", a.Binary, a.Name)); err != nil {
					return core.Failure(err, "Failed to start container"), err
				}
				return core.SuccessChange("Container started"), nil
			}
			if a.State == "stopped" && container.State.Running {
				if _, err := run(fmt.Sprintf("%s stop %s", a.Binary, a.Name)); err != nil {
					return core.Failure(err, "Failed to stop container"), err
				}
				return core.SuccessChange("Container stopped"), nil
			}
			return core.SuccessNoChange("Container already in desired state"), nil
		}
	}

	// Create (Run)
	if a.State == "running" || a.State == "stopped" {
		cmd := fmt.Sprintf("%s run -d --name %s", a.Binary, a.Name)

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

		if a.State == "stopped" {
			run(fmt.Sprintf("%s stop %s", a.Binary, a.Name))
			return core.SuccessChange("Container created (stopped)"), nil
		}

		return core.SuccessChange("Container started"), nil
	}

	return core.SuccessNoChange("No action needed"), nil
}
