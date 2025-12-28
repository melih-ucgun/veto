package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Resource struct {
	Type      string   `yaml:"type"`
	Name      string   `yaml:"name"`
	ID        string   `yaml:"id,omitempty"` // Kaynağın benzersiz adı
	Path      string   `yaml:"path,omitempty"`
	Content   string   `yaml:"content,omitempty"`
	State     string   `yaml:"state,omitempty"`
	Enabled   bool     `yaml:"enabled,omitempty"`
	DependsOn []string `yaml:"depends_on,omitempty"` // Bağımlı olduğu ID'lerin listesi
}

type Host struct {
	Name     string `yaml:"name"`
	Address  string `yaml:"address"` // Örn: 192.168.1.100:22
	User     string `yaml:"user"`
	Password string `yaml:"password,omitempty"` // Şimdilik basit tutuyoruz
}

type Config struct {
	Vars      map[string]interface{} `yaml:"vars,omitempty"` // Global değişkenler
	Resources []Resource             `yaml:"resources"`
	Hosts     []Host                 `yaml:"hosts,omitempty"`
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
