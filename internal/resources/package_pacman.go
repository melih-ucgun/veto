package resources

import "os/exec"

type PacmanProvider struct{}

func (p *PacmanProvider) IsInstalled(name string) (bool, error) {
	// Mevcut kodunuzdaki pacman -Q mantığı buraya gelir
	err := exec.Command("pacman", "-Q", name).Run()
	return err == nil, nil
}

func (p *PacmanProvider) Install(name string) error {
	// sudo pacman -S mantığı buraya gelir
	return exec.Command("sudo", "pacman", "-S", "--noconfirm", name).Run()
}
