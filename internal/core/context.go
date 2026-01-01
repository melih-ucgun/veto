package core

import (
	"context"
	"io"
	"os"
)

// SystemContext, uygulamanın çalışma anındaki bağlamını (context) tutar.
// Standart Go "context" paketini sarmalar ve Veto'a özel alanlar ekler.
type SystemContext struct {
	context.Context `yaml:"-"`

	// İşletim Sistemi Bilgileri
	OS         string `yaml:"os"`          // runtime.GOOS (linux, darwin)
	Kernel     string `yaml:"kernel"`      // 6.6.7-arch1-1
	Distro     string `yaml:"distro"`      // ubuntu, arch, fedora
	Version    string `yaml:"version"`     // 22.04, 38, rolling
	InitSystem string `yaml:"init_system"` // systemd, openrc, sysvinit
	Hostname   string `yaml:"hostname"`    // Makine adı

	// Donanım Bilgileri
	Hardware SystemHardware `yaml:"hardware"`

	// Çevresel Değişkenler
	Env SystemEnv `yaml:"env"`

	// Dosya Sistemi Metadatası
	FSInfo SystemFS `yaml:"fs"`

	// Dosya Sistemi Soyutlaması (İşlemler)
	FS FileSystem `yaml:"-"`

	// Taşıma Katmanı (Yerel veya Uzak)
	Transport Transport `yaml:"-"`

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

func NewSystemContext(dryRun bool) *SystemContext {
	return &SystemContext{
		Context:    context.Background(),
		OS:         "unknown",
		Distro:     "unknown",
		InitSystem: "unknown",
		User:       os.Getenv("USER"),
		HomeDir:    os.Getenv("HOME"),
		DryRun:     dryRun,
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
		FS:         &RealFS{}, // Default to local filesystem
		Transport:  &LocalTransport{},
		// Diğer alt structlar zero-value olarak başlar, detector tarafından doldurulur.
	}
}
