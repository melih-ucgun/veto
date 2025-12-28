package resources

import (
	"fmt"

	"github.com/melih-ucgun/monarch/internal/config"
)

// New, konfigürasyondaki ham veriyi alır ve ilgili Resource nesnesini oluşturur.
// Bu sürümde Global Templating desteği eklenmiştir: Name, Path, ID ve Content alanları
// vars (değişkenler) ile işlenir.
func New(r config.Resource, vars map[string]interface{}) (Resource, error) {
	// 1. Şablonlanabilir tüm alanları bir harita üzerinden döngüyle işle
	// r bir değer (value) olarak geldiği için üzerinde yaptığımız değişiklikler
	// bu fonksiyonun kapsamı ile sınırlıdır ve güvenlidir.
	fieldsToProcess := map[string]*string{
		"name":    &r.Name,
		"path":    &r.Path,
		"id":      &r.ID,
		"content": &r.Content,
	}

	for fieldName, fieldValue := range fieldsToProcess {
		if *fieldValue != "" {
			processed, err := config.ExecuteTemplate(*fieldValue, vars)
			if err != nil {
				return nil, fmt.Errorf("[%s] '%s' alanı şablon işleme hatası: %w", r.Name, fieldName, err)
			}
			*fieldValue = processed
		}
	}

	// 2. İşlenmiş (processed) verilerle Resource tipine göre nesneyi oluştur
	switch r.Type {
	case "file":
		return &FileResource{
			ResourceName: r.Name,
			Path:         r.Path,
			Content:      r.Content,
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
