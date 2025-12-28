package resources

import (
	"fmt"

	"github.com/melih-ucgun/monarch/internal/config"
)

// New, konfigürasyondaki ham veriyi alır ve ilgili Resource nesnesini oluşturur.
// Şablon (template) işleme mantığını da burada merkezileştiriyoruz.
func New(r config.Resource, vars map[string]interface{}) (Resource, error) {
	// 1. Şablonu işle (eğer içerik varsa)
	processedContent := r.Content
	if r.Content != "" {
		var err error
		processedContent, err = config.ExecuteTemplate(r.Content, vars)
		if err != nil {
			return nil, fmt.Errorf("[%s] şablon işleme hatası: %w", r.Name, err)
		}
	}

	// 2. Resource tipine göre nesneyi oluştur
	switch r.Type {
	case "file":
		return &FileResource{
			ResourceName: r.Name,
			Path:         r.Path,
			Content:      processedContent,
		}, nil

	case "package":
		return &PackageResource{
			PackageName: r.Name,
			State:       r.State,
			Provider:    GetDefaultProvider(),
		}, nil

	case "service":
		return &ServiceResource{
			ServiceName:  r.Name,
			DesiredState: r.State,
			Enabled:      r.Enabled,
		}, nil

	case "noop":
		return nil, nil // İşlem yapılmayacak, hata değil.

	default:
		return nil, fmt.Errorf("bilinmeyen kaynak tipi: %s", r.Type)
	}
}
