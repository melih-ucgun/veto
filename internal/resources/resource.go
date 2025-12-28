package resources

type Resource interface {
	ID() string
	Check() (bool, error) // İstenen durumda mı?
	Apply() error         // Durumu düzelt
}
