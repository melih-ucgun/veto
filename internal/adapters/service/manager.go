package service

// ServiceManager, init sistemleri (systemd, openrc, vb.) için ortak arayüzdür.
type ServiceManager interface {
	Name() string // systemd, openrc

	// State Checks
	IsEnabled(service string) (bool, error)
	IsActive(service string) (bool, error)

	// Actions
	Enable(service string) error
	Disable(service string) error
	Start(service string) error
	Stop(service string) error
	Restart(service string) error
	Reload(service string) error

	// Discovery
	ListEnabled() ([]string, error)
}
