package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// Config represents the root structure of monarch.yaml.
type Config struct {
	Vars      map[string]string `yaml:"vars"`      // Global variables
	Includes  []string          `yaml:"includes"`  // Other config files to include
	Resources []ResourceConfig  `yaml:"resources"` // Resource list
	Hosts     []Host            `yaml:"hosts"`     // Remote hosts (Optional)
}

// ResourceConfig holds the configuration for each resource (file, user, package, etc.).
type ResourceConfig struct {
	ID        string                 `yaml:"id"`
	Name      string                 `yaml:"name"`
	Type      string                 `yaml:"type"`
	State     string                 `yaml:"state"`
	When      string                 `yaml:"when"` // Conditional execution logic
	DependsOn []string               `yaml:"depends_on"`
	Params    map[string]interface{} `yaml:"params"`
}

// Host holds connection information for a remote host.
type Host struct {
	Name           string `yaml:"name"`
	Address        string `yaml:"address"`
	User           string `yaml:"user"`
	Port           int    `yaml:"port"`
	SSHKeyPath     string `yaml:"ssh_key_path"`
	BecomeMethod   string `yaml:"become_method"`   // sudo, su
	BecomePassword string `yaml:"become_password"` // Optional (Recommended to come from Vault)
}

// LoadConfig reads the YAML file at the specified path and converts it into a Config struct.
func LoadConfig(path string) (*Config, error) {
	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	// 0. If .env exists in project root or next to config, load it
	// Search next to config file
	baseDir := filepath.Dir(absPath)
	envPath := filepath.Join(baseDir, ".env")
	if _, err := os.Stat(envPath); err == nil {
		// .env found, load. Log if error but don't stop.
		if loadErr := godotenv.Load(envPath); loadErr != nil && !os.IsNotExist(loadErr) {
			// Just verify info, no logger yet
			fmt.Printf("Warning: Failed to load .env file: %v\n", loadErr)
		}
	} else {
		// Maybe in parent dir? (Project root)
		// Currently only checks next to config.
		// Alternative: godotenv.Load() without params searches in working dir.
		// Let's try this too:
		_ = godotenv.Load() // Ignore error (if no file found)
	}

	visited := make(map[string]bool)
	cfg, err := loadConfigRecursive(absPath, visited)
	if err != nil {
		return nil, err
	}

	// Recursive loading finished, now perform variable expansion on all string values
	expandConfig(cfg)

	return cfg, nil
}

func loadConfigRecursive(path string, visited map[string]bool) (*Config, error) {
	if visited[path] {
		return &Config{}, nil
	}
	visited[path] = true

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("file read error (%s): %w", path, err)
	}

	if len(data) == 0 {
		return &Config{}, nil
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("yaml parse error (%s): %w", path, err)
	}

	blockCfg := &cfg

	// Variable Expansion for path resolution before recursing
	// (If env var exists in includes field)
	for i, inc := range blockCfg.Includes {
		blockCfg.Includes[i] = os.ExpandEnv(inc)
	}

	// Process included files
	baseDir := filepath.Dir(path)
	var allResources []ResourceConfig

	// Include
	for _, includePath := range blockCfg.Includes {
		fullIncludePath := filepath.Join(baseDir, includePath)
		absIncludePath, err := filepath.Abs(fullIncludePath)
		if err != nil {
			return nil, err
		}

		subCfg, err := loadConfigRecursive(absIncludePath, visited)
		if err != nil {
			return nil, err
		}

		allResources = append(allResources, subCfg.Resources...)

		if blockCfg.Vars == nil {
			blockCfg.Vars = make(map[string]string)
		}
		for k, v := range subCfg.Vars {
			if _, exists := blockCfg.Vars[k]; !exists {
				blockCfg.Vars[k] = v
			}
		}
	}

	allResources = append(allResources, blockCfg.Resources...)
	blockCfg.Resources = allResources

	return blockCfg, nil
}

// expandConfig performs Env Var substitution on all string values in the configuration.
func expandConfig(cfg *Config) {
	// 1. Global Vars
	for k, v := range cfg.Vars {
		expanded := os.ExpandEnv(v)
		cfg.Vars[k] = expanded
		// Add to environment so resources can use these variables
		os.Setenv(k, expanded)
	}

	// 2. Resources
	for i := range cfg.Resources {
		expandResource(&cfg.Resources[i])
	}

	// 3. Hosts
	for i := range cfg.Hosts {
		cfg.Hosts[i].Address = os.ExpandEnv(cfg.Hosts[i].Address)
		cfg.Hosts[i].User = os.ExpandEnv(cfg.Hosts[i].User)
		cfg.Hosts[i].BecomePassword = os.ExpandEnv(cfg.Hosts[i].BecomePassword)
		cfg.Hosts[i].SSHKeyPath = os.ExpandEnv(cfg.Hosts[i].SSHKeyPath)
	}
}

func expandResource(res *ResourceConfig) {
	res.Name = os.ExpandEnv(res.Name)
	// ID does not change, references might break.
	// Type and State can also be expanded
	res.Type = os.ExpandEnv(res.Type)
	res.State = os.ExpandEnv(res.State)

	// Params (Recursive Map traversal)
	expandMap(res.Params)
}

func expandMap(m map[string]interface{}) {
	for k, v := range m {
		switch val := v.(type) {
		case string:
			m[k] = os.ExpandEnv(val)
		case map[string]interface{}:
			expandMap(val)
		case []interface{}:
			for i, item := range val {
				if str, ok := item.(string); ok {
					val[i] = os.ExpandEnv(str)
				} else if subMap, ok := item.(map[string]interface{}); ok {
					expandMap(subMap)
				}
			}
		}
	}
}
