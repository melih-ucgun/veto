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
	State     string   `yaml:"state,omitempty"` // installed, running, stopped, absent
	Enabled   bool     `yaml:"enabled,omitempty"`
	DependsOn []string `yaml:"depends_on,omitempty"`
	URL       string   `yaml:"url,omitempty"`
	Command   string   `yaml:"command,omitempty"`

	// Meta Veriler (İzinler ve Sahiplik)
	Mode  string `yaml:"mode,omitempty"`  // Örn: "0644"
	Owner string `yaml:"owner,omitempty"` // Örn: "root" veya "1000"
	Group string `yaml:"group,omitempty"` // Örn: "docker" veya "1000"

	// Symlink ve File Spesifik Alanlar
	Target string `yaml:"target,omitempty"`

	// Konteyner Spesifik Alanlar
	Image   string   `yaml:"image,omitempty"`
	Ports   []string `yaml:"ports,omitempty"`
	Env     []string `yaml:"env,omitempty"`
	Volumes []string `yaml:"volumes,omitempty"`
}

func (r *Resource) Identify() string {
	if r.ID != "" {
		return r.ID
	}
	return fmt.Sprintf("%s:%s", r.Type, r.Name)
}

type Host struct {
	Name           string `yaml:"name"`
	Address        string `yaml:"address"`
	User           string `yaml:"user"`
	Password       string `yaml:"password,omitempty"`
	KeyPath        string `yaml:"key_path,omitempty"`
	Passphrase     string `yaml:"passphrase,omitempty"`
	BecomePassword string `yaml:"become_password,omitempty"`
}

type Config struct {
	Vars      map[string]interface{} `yaml:"vars,omitempty"`
	Resources []Resource             `yaml:"resources"`
	Hosts     []Host                 `yaml:"hosts,omitempty"`
}

func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("konfigürasyon dosyası açılamadı: %w", err)
	}
	defer file.Close()

	var cfg Config
	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)

	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("YAML ayrıştırma hatası (şemaya uymuyor): %w", err)
	}

	if err := cfg.ResolveSecrets(); err != nil {
		return nil, fmt.Errorf("sırlar çözülemedi: %w", err)
	}

	return &cfg, nil
}

func (c *Config) ResolveSecrets() error {
	privKey := os.Getenv("MONARCH_KEY")
	if privKey == "" {
		return nil
	}

	for k, v := range c.Vars {
		if strVal, ok := v.(string); ok && strings.HasPrefix(strVal, "-----BEGIN AGE ENCRYPTED FILE-----") {
			dec, err := crypto.Decrypt(strVal, privKey)
			if err == nil {
				c.Vars[k] = dec
			}
		}
	}

	for i := range c.Hosts {
		if strings.HasPrefix(c.Hosts[i].BecomePassword, "-----BEGIN AGE ENCRYPTED FILE-----") {
			dec, err := crypto.Decrypt(c.Hosts[i].BecomePassword, privKey)
			if err == nil {
				c.Hosts[i].BecomePassword = dec
			}
		}
	}
	return nil
}
