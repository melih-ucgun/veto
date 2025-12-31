package discovery

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

func discoverServices(initSystem string) ([]string, error) {
	switch initSystem {
	case "systemd":
		return discoverSystemd()
	case "openrc":
		return discoverOpenRC()
	case "sysvinit":
		return discoverSysVinit()
	default:
		return nil, fmt.Errorf("unsupported init system: %s", initSystem)
	}
}

func discoverSystemd() ([]string, error) {
	// systemctl list-unit-files --state=enabled --type=service --no-legend
	cmd := exec.Command("systemctl", "list-unit-files", "--state=enabled", "--type=service", "--no-legend")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var services []string
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) > 0 {
			svc := fields[0]
			// remove .service suffix
			svc = strings.TrimSuffix(svc, ".service")
			services = append(services, svc)
		}
	}
	return services, nil
}

func discoverOpenRC() ([]string, error) {
	// rc-update show default
	cmd := exec.Command("rc-update", "show", "default")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var services []string
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) > 0 {
			services = append(services, fields[0])
		}
	}
	return services, nil
}

func discoverSysVinit() ([]string, error) {
	// This is messy across distros.
	// Debian/Ubuntu: service --status-all | grep +
	// But sysvinit is rare now. Returning empty for now.
	return []string{}, nil
}
