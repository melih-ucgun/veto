package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
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

	// 0. Proje kökünde veya config yanında .env varsa yükle
	// Config dosyasının yanında ara
	baseDir := filepath.Dir(absPath)
	envPath := filepath.Join(baseDir, ".env")
	if _, err := os.Stat(envPath); err == nil {
		// .env bulundu, yükle. Hata varsa logla ama durma.
		if loadErr := godotenv.Load(envPath); loadErr != nil && !os.IsNotExist(loadErr) {
			// Sadece info geçebiliriz, henüz logger yok
			fmt.Printf("Warning: Failed to load .env file: %v\n", loadErr)
		}
	} else {
		// Belki bir üst dizinde? (Proje root)
		// Şimdilik sadece config yanında bakar.
		// Alternatif: godotenv.Load() parametresiz çağrılırsa working dir'de arar.
		// Bunu da deneyelim:
		_ = godotenv.Load() // Ignore error (if no file found)
	}

	visited := make(map[string]bool)
	cfg, err := loadConfigRecursive(absPath, visited)
	if err != nil {
		return nil, err
	}

	// Recursive yükleme bitti, şimdi tüm string değerlerde variable expansion yap
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
		return nil, fmt.Errorf("dosya okuma hatası (%s): %w", path, err)
	}

	if len(data) == 0 {
		return &Config{}, nil
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("yaml parse hatası (%s): %w", path, err)
	}

	blockCfg := &cfg

	// Variable Expansion for path resolution before recursing
	// (Eğer includes alanında env var varsa)
	for i, inc := range blockCfg.Includes {
		blockCfg.Includes[i] = os.ExpandEnv(inc)
	}

	// Include edilenleri işle
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

// expandConfig tüm konfigürasyondaki string değerlerde Env Var substitution yapar.
func expandConfig(cfg *Config) {
	// 1. Global Vars
	for k, v := range cfg.Vars {
		expanded := os.ExpandEnv(v)
		cfg.Vars[k] = expanded
		// Kaynaklar bu değişkenleri kullanabilsin diye environment'a ekle
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
	// ID değişmez, referanslar bozulabilir.
	// Type ve State de expand edilebilir
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
