package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Resource struct {
	Type string `yaml:"type"`
	Name string `yaml:"name"`
}

type Config struct {
	Resources []Resource `yaml:"resources"` // tag küçük harf, field büyük harf
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
