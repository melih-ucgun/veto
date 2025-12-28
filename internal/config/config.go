package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/melih-ucgun/monarch/internal/crypto"
	"gopkg.in/yaml.v3"
)

type Resource struct {
	Type      string   `yaml:"type"`
	Name      string   `yaml:"name"`
	ID        string   `yaml:"id,omitempty"`
	Path      string   `yaml:"path,omitempty"`
	Content   string   `yaml:"content,omitempty"`
	State     string   `yaml:"state,omitempty"`
	Enabled   bool     `yaml:"enabled,omitempty"`
	DependsOn []string `yaml:"depends_on,omitempty"`
	URL       string   `yaml:"url,omitempty"`
}

type Host struct {
	Name       string `yaml:"name"`
	Address    string `yaml:"address"`
	User       string `yaml:"user"`
	Password   string `yaml:"password,omitempty"`
	KeyPath    string `yaml:"key_path,omitempty"`
	Passphrase string `yaml:"passphrase,omitempty"`
}

type Config struct {
	Vars      map[string]interface{} `yaml:"vars,omitempty"`
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

	// Sırları çöz
	if err := cfg.ResolveSecrets(); err != nil {
		return nil, fmt.Errorf("sırlar çözülemedi: %w", err)
	}

	return &cfg, nil
}

// ResolveSecrets, vars içindeki şifreli değerleri (secret://...) bulur ve çözer.
func (c *Config) ResolveSecrets() error {
	privKey := os.Getenv("MONARCH_KEY")
	if privKey == "" {
		// Eğer ortam değişkeni yoksa şifreli değerleri atla veya uyarı ver
		return nil
	}

	for k, v := range c.Vars {
		strVal, ok := v.(string)
		if !ok {
			continue
		}

		// Eğer değer "secret:" ile başlıyorsa çözmeye çalış
		if strings.HasPrefix(strVal, "-----BEGIN AGE ENCRYPTED FILE-----") {
			decrypted, err := crypto.Decrypt(strVal, privKey)
			if err != nil {
				return fmt.Errorf("değişken '%s' çözülemedi: %v", k, err)
			}
			c.Vars[k] = decrypted
		}
	}
	return nil
}
