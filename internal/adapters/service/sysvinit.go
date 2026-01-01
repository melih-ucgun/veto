package service

import (
	"fmt"
	"os/exec"
	"strings"
)

type SysVinitManager struct {
	enableCmd string // "update-rc.d" or "chkconfig"
}

func NewSysVinitManager() *SysVinitManager {
	mgr := &SysVinitManager{}
	if _, err := exec.LookPath("update-rc.d"); err == nil {
		mgr.enableCmd = "update-rc.d"
	} else if _, err := exec.LookPath("chkconfig"); err == nil {
		mgr.enableCmd = "chkconfig"
	}
	return mgr
}

func (s *SysVinitManager) Name() string {
	return "sysvinit"
}

func (s *SysVinitManager) IsEnabled(service string) (bool, error) {
	if s.enableCmd == "chkconfig" {
		cmd := exec.Command("chkconfig", "--list", service)
		out, _ := cmd.CombinedOutput()
		return strings.Contains(string(out), ":on"), nil
	}
	// update-rc.d doesn't easily show status, assuming false or skipping
	// ls /etc/rc3.d/S*service
	cmd := exec.Command("bash", "-c", fmt.Sprintf("ls /etc/rc*.d/S*%s 2>/dev/null", service))
	err := cmd.Run()
	return err == nil, nil
}

func (s *SysVinitManager) IsActive(service string) (bool, error) {
	cmd := exec.Command("service", service, "status")
	err := cmd.Run()
	return err == nil, nil
}

func (s *SysVinitManager) Enable(service string) error {
	if s.enableCmd == "chkconfig" {
		return s.run("chkconfig", service, "on")
	} else if s.enableCmd == "update-rc.d" {
		return s.run("update-rc.d", service, "defaults")
	}
	return fmt.Errorf("no enable command found for sysvinit")
}

func (s *SysVinitManager) Disable(service string) error {
	if s.enableCmd == "chkconfig" {
		return s.run("chkconfig", service, "off")
	} else if s.enableCmd == "update-rc.d" {
		return s.run("update-rc.d", service, "remove")
	}
	return fmt.Errorf("no disable command found for sysvinit")
}

func (s *SysVinitManager) Start(service string) error {
	return s.runService(service, "start")
}

func (s *SysVinitManager) Stop(service string) error {
	return s.runService(service, "stop")
}

func (s *SysVinitManager) Restart(service string) error {
	return s.runService(service, "restart")
}

func (s *SysVinitManager) Reload(service string) error {
	return s.runService(service, "reload")
}

func (s *SysVinitManager) runService(service string, action string) error {
	cmd := exec.Command("service", service, action)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("service %s %s failed: %s: %w", service, action, string(out), err)
	}
	return nil
}

func (s *SysVinitManager) run(cmdName string, args ...string) error {
	cmd := exec.Command(cmdName, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s %s failed: %s: %w", cmdName, strings.Join(args, " "), string(out), err)
	}
	return nil
}
func (s *SysVinitManager) ListEnabled() ([]string, error) {
	return nil, fmt.Errorf("list enabled services is not supported for SysVinit yet")
}
