package resources

import "fmt"

type PackageManager interface {
	IsInstalled(name string) (bool, error)
	Install(name string) error
	Remove(name string) error
}

type PackageResource struct {
	CanonicalID string
	PackageName string
	State       string
	Provider    PackageManager
}

func (p *PackageResource) ID() string {
	return p.CanonicalID
}

func (p *PackageResource) Check() (bool, error) {
	return p.Provider.IsInstalled(p.PackageName)
}

func (p *PackageResource) Diff() (string, error) {
	installed, _ := p.Check()
	if p.State == "absent" && installed {
		return fmt.Sprintf("- package: %s (kaldırılacak)", p.PackageName), nil
	}
	if p.State != "absent" && !installed {
		return fmt.Sprintf("+ package: %s (kurulacak)", p.PackageName), nil
	}
	return "", nil
}

func (p *PackageResource) Apply() error {
	if p.State == "absent" {
		return p.Provider.Remove(p.PackageName)
	}
	return p.Provider.Install(p.PackageName)
}
