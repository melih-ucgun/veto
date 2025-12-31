package core

import (
	"context"
	"io"
	"os"
)

// SystemContext, uygulamanın çalışma anındaki bağlamını (context) tutar.
// Standart Go "context" paketini sarmalar ve Monarch'a özel alanlar ekler.
type SystemContext struct {
	context.Context

	// İşletim Sistemi Bilgileri
	OS       string `yaml:"os"`       // runtime.GOOS (linux, darwin)
	Distro   string `yaml:"distro"`   // ubuntu, arch, fedora
	Version  string `yaml:"version"`  // 22.04, 38, rolling
	Hostname string `yaml:"hostname"` // Makine adı

	// Donanım Bilgileri
	Hardware SystemHardware `yaml:"hardware"`

	// Çevresel Değişkenler
	Env SystemEnv `yaml:"env"`

	// Dosya Sistemi
	FS SystemFS `yaml:"fs"`

	// Kullanıcı Bilgileri
	User    string `yaml:"user"`     // Mevcut kullanıcı
	HomeDir string `yaml:"home_dir"` // Kullanıcının ev dizini
	UID     string `yaml:"uid"`      // User ID
	GID     string `yaml:"gid"`      // Group ID

	// Çalışma Modu
	DryRun bool `yaml:"-"` // Eğer true ise, hiçbir değişiklik yapılmaz, sadece simüle edilir.

	// Logger veya Output (İleride loglama için)
	Stdout io.Writer `yaml:"-"`
	Stderr io.Writer `yaml:"-"`
}

type SystemHardware struct {
	CPUModel  string `yaml:"cpu_model"`  // "AMD Ryzen 7 5800X"
	CPUCore   int    `yaml:"cpu_core"`   // Çekirdek sayısı
	RAMTotal  string `yaml:"ram_total"`  // "16GB"
	GPUVendor string `yaml:"gpu_vendor"` // "NVIDIA", "AMD", "Intel"
	GPUModel  string `yaml:"gpu_model"`  // "RTX 3070"
}

type SystemEnv struct {
	Shell    string `yaml:"shell"`    // "/bin/zsh"
	Lang     string `yaml:"lang"`     // "en_US.UTF-8"
	Term     string `yaml:"term"`     // "xterm-256color"
	Timezone string `yaml:"timezone"` // "Europe/Istanbul"
}

type SystemFS struct {
	RootFSType string `yaml:"root_fs_type"` // "ext4", "btrfs", "zfs"
}

// NewSystemContext, temel bir context oluşturur.
func NewSystemContext(dryRun bool) *SystemContext {
	return &SystemContext{
		Context: context.Background(),
		OS:      "unknown",
		Distro:  "unknown",
		User:    os.Getenv("USER"),
		HomeDir: os.Getenv("HOME"),
		DryRun:  dryRun,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		// Diğer alt structlar zero-value olarak başlar, detector tarafından doldurulur.
	}
}
