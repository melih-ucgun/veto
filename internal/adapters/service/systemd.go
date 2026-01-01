package service

import (
	"fmt"
	"os/exec"
	"strings"
)

type SystemdManager struct{}

func NewSystemdManager() *SystemdManager {
	return &SystemdManager{}
}

func (s *SystemdManager) Name() string {
	return "systemd"
}

func (s *SystemdManager) IsEnabled(service string) (bool, error) {
	cmd := exec.Command("systemctl", "is-enabled", service)
	out, _ := cmd.CombinedOutput() // Error is expected if disabled
	status := strings.TrimSpace(string(out))
	return status == "enabled", nil
}

func (s *SystemdManager) IsActive(service string) (bool, error) {
	cmd := exec.Command("systemctl", "is-active", service)
	out, _ := cmd.CombinedOutput() // Error is expected if inactive
	status := strings.TrimSpace(string(out))
	return status == "active", nil
}

func (s *SystemdManager) Enable(service string) error {
	return s.run("enable", "--now", service)
}

func (s *SystemdManager) Disable(service string) error {
	return s.run("disable", "--now", service)
}

func (s *SystemdManager) Start(service string) error {
	return s.run("start", service)
}

func (s *SystemdManager) Stop(service string) error {
	return s.run("stop", service)
}

func (s *SystemdManager) Restart(service string) error {
	return s.run("restart", service)
}

func (s *SystemdManager) Reload(service string) error {
	return s.run("reload", service)
}

func (s *SystemdManager) run(args ...string) error {
	cmd := exec.Command("systemctl", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl %s failed: %s: %w", strings.Join(args, " "), string(out), err)
	}
	return nil
}

func (s *SystemdManager) ListEnabled() ([]string, error) {
	// systemctl list-unit-files --state=enabled --type=service --no-legend --no-pager
	cmd := exec.Command("systemctl", "list-unit-files", "--state=enabled", "--type=service", "--no-legend", "--no-pager")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list enabled services: %w", err)
	}

	var services []string
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) > 0 {
			// fields[0] is service name (e.g., "sshd.service")
			services = append(services, fields[0])
		}
	}
	return services, nil
}
