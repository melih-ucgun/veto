package resources

import (
	"context"
	"fmt"
)

type PackageResource struct {
	CanonicalID string      `mapstructure:"-"`
	Name        interface{} `mapstructure:"name"`    // String veya Map[string]string olabilir
	ManagerName string      `mapstructure:"manager"` // pacman, dnf, apt vb.
	State       string      `mapstructure:"state"`
}

func (r *PackageResource) ID() string {
	return r.CanonicalID
}

// getActualName: Manager tipine göre doğru paket ismini döner
func (r *PackageResource) getActualName() (string, error) {
	switch v := r.Name.(type) {
	case string:
		return v, nil
	case map[string]interface{}:
		if name, ok := v[r.ManagerName].(string); ok {
			return name, nil
		}
		return "", fmt.Errorf("'%s' paket yöneticisi için bir isim belirtilmemiş", r.ManagerName)
	case map[interface{}]interface{}: // YAML parser bazen bu tipi dönebilir
		key := interface{}(r.ManagerName)
		if name, ok := v[key].(string); ok {
			return name, nil
		}
		return "", fmt.Errorf("'%s' paket yöneticisi için bir isim belirtilmemiş", r.ManagerName)
	default:
		return "", fmt.Errorf("geçersiz paket ismi formatı")
	}
}

func (r *PackageResource) Check() (bool, error) {
	packageName, err := r.getActualName()
	if err != nil {
		return false, err
	}

	mgr, err := GetPackageManager(r.ManagerName)
	if err != nil {
		return false, err
	}

	isInstalled, err := mgr.IsInstalled(packageName)
	if err != nil {
		return false, fmt.Errorf("paket durumu sorgulanamadı (%s): %w", packageName, err)
	}

	if r.State == "installed" {
		return isInstalled, nil
	}
	return !isInstalled, nil
}

func (r *PackageResource) Apply() error {
	packageName, err := r.getActualName()
	if err != nil {
		return err
	}

	mgr, err := GetPackageManager(r.ManagerName)
	if err != nil {
		return err
	}

	if r.State == "installed" {
		return mgr.Install(packageName)
	}
	return mgr.Remove(packageName)
}

func (r *PackageResource) Undo(ctx context.Context) error {
	packageName, err := r.getActualName()
	if err != nil {
		return err
	}

	mgr, err := GetPackageManager(r.ManagerName)
	if err != nil {
		return err
	}

	if r.State == "installed" {
		return mgr.Remove(packageName)
	}
	return mgr.Install(packageName)
}

func (r *PackageResource) Diff() (string, error) {
	name, _ := r.getActualName()
	return fmt.Sprintf("Package[%s] state mismatch (Manager: %s)", name, r.ManagerName), nil
}
