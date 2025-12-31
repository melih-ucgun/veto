package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config, monarch.yaml dosyasının kök yapısını temsil eder.
type Config struct {
	Vars      map[string]string `yaml:"vars"`      // Global değişkenler
	Includes  []string          `yaml:"includes"`  // Dahil edilecek diğer config dosyaları
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
	// Mutlak yol al
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	visited := make(map[string]bool)
	return loadConfigRecursive(absPath, visited)
}

func loadConfigRecursive(path string, visited map[string]bool) (*Config, error) {
	if visited[path] {
		// Döngüsel import (Circular dependency)
		// Şimdilik hata vermeyip, tekrar yüklemeyi reddedebiliriz veya hata verebiliriz.
		// Basitlik için dönelim.
		return &Config{}, nil
	}
	visited[path] = true

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("dosya okuma hatası (%s): %w", path, err)
	}

	if len(data) == 0 {
		return &Config{}, nil
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("yaml parse hatası (%s): %w", path, err)
	}

	// Include edilenleri işle
	baseDir := filepath.Dir(path)
	var allResources []ResourceConfig

	// Önce included dosyaları yükle (Öncelik sırası: Include edilenler -> Ana dosya)
	// Aslında kaynakların sırası DAG ile belirlendiği için import sırası çok fark etmez
	// ama değişken override mantığı için önemli olabilir.
	// Vars sayalım: Ana dosyadaki değişkenler override etmeli mi? Evet genelde main ezer.
	for _, includePath := range cfg.Includes {
		// Include path relative is kabul edilir
		fullIncludePath := filepath.Join(baseDir, includePath)
		absIncludePath, err := filepath.Abs(fullIncludePath)
		if err != nil {
			return nil, err
		}

		subCfg, err := loadConfigRecursive(absIncludePath, visited)
		if err != nil {
			return nil, err
		}

		// Resource'ları birleştir
		allResources = append(allResources, subCfg.Resources...)

		// TODO: Vars ve Hosts birleştirme de yapılmalı
		if cfg.Vars == nil {
			cfg.Vars = make(map[string]string)
		}
		for k, v := range subCfg.Vars {
			// Eğer ana dosyada yoksa ekle (ana dosya ezer)
			if _, exists := cfg.Vars[k]; !exists {
				cfg.Vars[k] = v
			}
		}
	}

	// Ana dosyanın kaynaklarını ekle
	allResources = append(allResources, cfg.Resources...)
	cfg.Resources = allResources

	return &cfg, nil
}
