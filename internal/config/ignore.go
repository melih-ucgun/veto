package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

type IgnoreManager struct {
	patterns []string
	filePath string
}

func NewIgnoreManager(path string) (*IgnoreManager, error) {
	mgr := &IgnoreManager{
		filePath: path,
		patterns: []string{},
	}

	// If file exists, load it. If not, we just have an empty list.
	if _, err := os.Stat(path); err == nil {
		if err := mgr.Load(); err != nil {
			return nil, err
		}
	}
	// We don't error if file missing, we treat it as empty.

	return mgr, nil
}

func (m *IgnoreManager) Load() error {
	file, err := os.Open(m.filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		m.patterns = append(m.patterns, line)
	}
	return scanner.Err()
}

func (m *IgnoreManager) IsIgnored(name string) bool {
	for _, pattern := range m.patterns {
		matched, _ := filepath.Match(pattern, name)
		if matched {
			return true
		}
		// Also check partial matching for paths if needed?
		// Gitignore logic effectively matches relative paths.
		// For now simple Glob match on the name/path string provided.
	}
	return false
}

func (m *IgnoreManager) Add(pattern string) error {
	// Check if already exists to avoid duplicates
	for _, p := range m.patterns {
		if p == pattern {
			return nil
		}
	}

	m.patterns = append(m.patterns, pattern)

	// Append to file
	f, err := os.OpenFile(m.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.WriteString(pattern + "\n"); err != nil {
		return err
	}
	return nil
}
