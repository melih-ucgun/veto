package core

// Resource, sistemdeki yönetilebilir birimi temsil eden arayüz.
// Artık Core paketinde olduğu için Import Cycle sorunu çözülüyor.
type Resource interface {
	Apply(ctx *SystemContext) (Result, error)
	Check(ctx *SystemContext) (bool, error)
	Validate() error
	GetName() string
}

// Revertable, geri alınabilir kaynakların implemente etmesi gereken arayüz.
type Revertable interface {
	Revert(ctx *SystemContext) error
}

// BaseResource, ortak alanları tutar.
type BaseResource struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
}

func (b *BaseResource) GetName() string {
	return b.Name
}
