package discovery

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/melih-ucgun/monarch/internal/core"
)

func discoverPackages(ctx *core.SystemContext) ([]string, error) {
	// Detect based on Distro or assume from context
	// Simple heuristic: check for binary existence

	// Order matters: check specific helpers (yay/paru) before generic (pacman)
	// check universal (flatpak/snap/brew) regardless of system pkg manager

	var allPkgs []string

	// 1. System Package Managers
	if isCommandAvailable("paru") {
		p, _ := discoverGeneric("paru", "-Qqe")
		allPkgs = append(allPkgs, p...)
	} else if isCommandAvailable("yay") {
		p, _ := discoverGeneric("yay", "-Qqe")
		allPkgs = append(allPkgs, p...)
	} else if isCommandAvailable("pacman") {
		p, _ := discoverGeneric("pacman", "-Qqe")
		allPkgs = append(allPkgs, p...)
	} else if isCommandAvailable("dnf") {
		p, _ := discoverGeneric("dnf", "repoquery", "--userinstalled", "--queryformat", "%{name}")
		allPkgs = append(allPkgs, p...)
	} else if isCommandAvailable("yum") {
		// yum doesn't support repoquery --userinstalled matching dnf perfectly in all versions,
		// but often implies dnf. Fallback to extracting names from 'yum list installed' is messy.
		// Try same as dnf or simple rpm
		p, _ := discoverGeneric("rpm", "-qa", "--qf", "%{NAME}\n")
		allPkgs = append(allPkgs, p...)
	} else if isCommandAvailable("zypper") {
		// zypper is hard to filter for "user installed". rpm -qa is safest fallback.
		p, _ := discoverGeneric("rpm", "-qa", "--qf", "%{NAME}\n")
		allPkgs = append(allPkgs, p...)
	} else if isCommandAvailable("apt") {
		p, _ := discoverGeneric("apt-mark", "showmanual")
		allPkgs = append(allPkgs, p...)
	} else if isCommandAvailable("apk") {
		p, _ := discoverGeneric("apk", "info")
		allPkgs = append(allPkgs, p...)
	}

	// 2. Extra Managers (can coexist)
	if isCommandAvailable("brew") {
		p, _ := discoverGeneric("brew", "leaves")
		allPkgs = append(allPkgs, p...)
	}
	if isCommandAvailable("flatpak") {
		p, _ := discoverGeneric("flatpak", "list", "--app", "--columns=application")
		allPkgs = append(allPkgs, p...)
	}
	if isCommandAvailable("snap") {
		// snap list outputs header, need to ignore it.
		// generic helper splits lines, we just need to skip first line?
		// Or assume awk usage on system. simpler to just read and skip line 0.
		if out, err := exec.Command("snap", "list").Output(); err == nil {
			lines := parseLines(out)
			if len(lines) > 0 {
				// skip header "Name  Version  Rev..."
				for _, line := range lines[1:] {
					fields := strings.Fields(line)
					if len(fields) > 0 {
						allPkgs = append(allPkgs, fields[0])
					}
				}
			}
		}
	}

	if len(allPkgs) == 0 {
		return nil, fmt.Errorf("no supported package manager found or no packages detected")
	}

	return unique(allPkgs), nil
}

func discoverGeneric(cmd string, args ...string) ([]string, error) {
	c := exec.Command(cmd, args...)
	output, err := c.Output()
	if err != nil {
		return nil, err
	}
	return parseLines(output), nil
}

func isCommandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func parseLines(data []byte) []string {
	var lines []string
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func unique(slice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range slice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}
