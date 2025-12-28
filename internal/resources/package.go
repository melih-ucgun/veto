package resources

// PackageManager, farklı işletim sistemlerindeki paket yöneticileri (pacman, apt, brew vb.)
// için ortak bir arayüz tanımlar.
type PackageManager interface {
	IsInstalled(name string) (bool, error)
	Install(name string) error
	Remove(name string) error
}

// PackageResource, bir paketin sistemdeki varlığını ve durumunu temsil eder.
type PackageResource struct {
	PackageName string
	State       string
	Provider    PackageManager
}

// ID, kaynağın benzersiz kimliğini döner (Örn: pkg:neovim).
func (p *PackageResource) ID() string {
	return "pkg:" + p.PackageName
}

// Check, paketin şu anki durumunu kontrol eder.
func (p *PackageResource) Check() (bool, error) {
	return p.Provider.IsInstalled(p.PackageName)
}

// Apply, paketi istenen duruma (kurulu veya silinmiş) getirir.
func (p *PackageResource) Apply() error {
	if p.State == "absent" {
		return p.Provider.Remove(p.PackageName)
	}
	return p.Provider.Install(p.PackageName)
}
