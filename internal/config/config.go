package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Vars      map[string]interface{} `yaml:"vars"`
	Hosts     []Host                 `yaml:"hosts"`
	Resources []Resource             `yaml:"resources"`
}

type Host struct {
	Name           string `yaml:"name"`
	Address        string `yaml:"address"`
	Port           int    `yaml:"port"`
	User           string `yaml:"user"`
	Password       string `yaml:"password"`
	BecomePassword string `yaml:"become_password"`
}

type Resource struct {
	Name      string            `yaml:"name"`
	Type      string            `yaml:"type"`
	Path      string            `yaml:"path"`
	Content   string            `yaml:"content"`
	Source    string            `yaml:"source"`
	Target    string            `yaml:"target"`
	Mode      string            `yaml:"mode"`
	Owner     string            `yaml:"owner"`
	Group     string            `yaml:"group"`
	State     string            `yaml:"state"` // present, absent, started, stopped
	Enabled   bool              `yaml:"enabled"`
	Image     string            `yaml:"image"`
	Ports     []string          `yaml:"ports"`
	Env       map[string]string `yaml:"env"`
	Volumes   []string          `yaml:"volumes"`
	Command   string            `yaml:"command"`
	Creates   string            `yaml:"creates"`
	OnlyIf    string            `yaml:"only_if"`
	Unless    string            `yaml:"unless"`
	URL       string            `yaml:"url"`
	DependsOn []string          `yaml:"depends_on"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Identify, kaynağı tekilleştirmek için bir ID üretir.
func (r *Resource) Identify() string {
	return r.Type + ":" + r.Name
}
