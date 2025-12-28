package resources

// Resource, Monarch tarafından yönetilen her varlığın uyması gereken arayüzdür.
type Resource interface {
	ID() string
	Check() (bool, error)  // İstenen durumda mı?
	Apply() error          // Durumu düzelt
	Diff() (string, error) // Mevcut ve istenen durum arasındaki farkı metin olarak döner
}
