package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config, monarch.yaml dosyasının kök yapısını temsil eder.
type Config struct {
	Vars      map[string]string `yaml:"vars"`      // Global değişkenler
	Resources []ResourceConfig  `yaml:"resources"` // Kaynak listesi
	Hosts     []Host            `yaml:"hosts"`     // Uzak sunucular (Opsiyonel)
}

// ResourceConfig, her bir kaynağın (file, user, package vb.) konfigürasyonunu tutar.
type ResourceConfig struct {
	ID        string                 `yaml:"id"`
	Name      string                 `yaml:"name"`
	Type      string                 `yaml:"type"`
	State     string                 `yaml:"state"`
	DependsOn []string               `yaml:"depends_on"`
	Params    map[string]interface{} `yaml:"parameters"`
}

// Host, uzak sunucu bağlantı bilgilerini tutar.
type Host struct {
	Name           string `yaml:"name"`
	Address        string `yaml:"address"`
	User           string `yaml:"user"`
	Port           int    `yaml:"port"`
	SSHKeyPath     string `yaml:"ssh_key_path"`
	BecomeMethod   string `yaml:"become_method"`   // sudo, su
	BecomePassword string `yaml:"become_password"` // Opsiyonel (Vault'tan gelmesi önerilir)
}

// LoadConfig, belirtilen yoldaki YAML dosyasını okur ve Config struct'ına çevirir.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("dosya okuma hatası: %w", err)
	}

	// Eğer dosya boşsa veya sadece yorum varsa hata vermemeli, boş config dönmeli
	if len(data) == 0 {
		return &Config{}, nil
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("yaml parse hatası: %w", err)
	}

	return &cfg, nil
}
