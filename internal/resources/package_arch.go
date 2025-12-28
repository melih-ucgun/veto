package resources

import (
	"fmt"
	"os/exec"
)

// ArchLinuxProvider; pacman, yay ve paru gibi araçları tek bir yapıdan yöneten adaptördür.
type ArchLinuxProvider struct {
	Binary  string // Kullanılacak komut (pacman, yay, paru)
	UseSudo bool   // Komutun sudo ile çalıştırılıp çalıştırılmayacağı
}

func (p *ArchLinuxProvider) IsInstalled(name string) (bool, error) {
	// -Q flag'i paketin kurulu olup olmadığını hızlıca kontrol eder.
	err := exec.Command(p.Binary, "-Q", name).Run()
	return err == nil, nil
}

func (p *ArchLinuxProvider) Install(name string) error {
	args := []string{"-S", "--noconfirm", name}
	cmdName := p.Binary

	if p.UseSudo {
		args = append([]string{p.Binary}, args...)
		cmdName = "sudo"
	}

	return exec.Command(cmdName, args...).Run()
}

func (p *ArchLinuxProvider) Remove(name string) error {
	args := []string{"-R", "--noconfirm", name}
	cmdName := p.Binary

	if p.UseSudo {
		args = append([]string{p.Binary}, args...)
		cmdName = "sudo"
	}

	return exec.Command(cmdName, args...).Run()
}

// GetDefaultProvider, sistemdeki paket yöneticisini tespit eder ve uygun provider'ı döner.
func GetDefaultProvider() PackageManager {
	// Öncelik: AUR Helperlar -> Native Pacman
	if _, err := exec.LookPath("paru"); err == nil {
		return &ArchLinuxProvider{Binary: "paru", UseSudo: false}
	}
	if _, err := exec.LookPath("yay"); err == nil {
		return &ArchLinuxProvider{Binary: "yay", UseSudo: false}
	}
	if _, err := exec.LookPath("pacman"); err == nil {
		return &ArchLinuxProvider{Binary: "pacman", UseSudo: true}
	}

	fmt.Println("❌ Uyarı: Sistemde desteklenen bir Arch paket yöneticisi bulunamadı.")
	return nil
}
