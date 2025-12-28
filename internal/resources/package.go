package resources

type PackageManager interface {
	IsInstalled(name string) (bool, error)
	Install(name string) error
	Remove(name string) error // Gelecek için "absent" durumu
}

type PackageResource struct {
	PackageName string
	State       string         // "installed" veya "absent"
	Provider    PackageManager // Modülerliği sağlayan kısım
}

func (p *PackageResource) Check() (bool, error) {
	return p.Provider.IsInstalled(p.PackageName)
}

func (p *PackageResource) Apply() error {
	if p.State == "installed" || p.State == "" {
		return p.Provider.Install(p.PackageName)
	}
	return nil
}
