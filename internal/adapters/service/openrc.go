package service

import (
	"fmt"
	"os/exec"
	"strings"
)

type OpenRCManager struct{}

func NewOpenRCManager() *OpenRCManager {
	return &OpenRCManager{}
}

func (s *OpenRCManager) Name() string {
	return "openrc"
}

func (s *OpenRCManager) IsEnabled(service string) (bool, error) {
	// rc-update show default | grep service
	// This is a bit loose, but standard for OpenRC
	cmd := exec.Command("rc-update", "show", "default")
	out, _ := cmd.CombinedOutput()
	return strings.Contains(string(out), service), nil
}

func (s *OpenRCManager) IsActive(service string) (bool, error) {
	cmd := exec.Command("rc-service", service, "status")
	err := cmd.Run()
	// rc-service returns 0 if started, non-zero if stopped
	return err == nil, nil
}

func (s *OpenRCManager) Enable(service string) error {
	return s.runUpdate("add", service, "default")
}

func (s *OpenRCManager) Disable(service string) error {
	return s.runUpdate("del", service, "default")
}

func (s *OpenRCManager) Start(service string) error {
	return s.runService(service, "start")
}

func (s *OpenRCManager) Stop(service string) error {
	return s.runService(service, "stop")
}

func (s *OpenRCManager) Restart(service string) error {
	return s.runService(service, "restart")
}

func (s *OpenRCManager) Reload(service string) error {
	return s.runService(service, "reload")
}

func (s *OpenRCManager) runService(service string, action string) error {
	cmd := exec.Command("rc-service", service, action)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("rc-service %s %s failed: %s: %w", service, action, string(out), err)
	}
	return nil
}

func (s *OpenRCManager) runUpdate(action, service, runlevel string) error {
	cmd := exec.Command("rc-update", action, service, runlevel)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("rc-update %s %s %s failed: %s: %w", action, service, runlevel, string(out), err)
	}
	return nil
}
func (s *OpenRCManager) ListEnabled() ([]string, error) {
	return nil, fmt.Errorf("list enabled services is not supported for OpenRC yet")
}
