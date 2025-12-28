package resources

import (
	"fmt"

	"github.com/melih-ucgun/monarch/internal/config"
)

// New, konfigürasyondan gelen veriyi işleyerek somut bir Resource nesnesi oluşturur.
func New(r config.Resource, vars map[string]interface{}) (Resource, error) {
	// İşlenecek alanları bir haritada topluyoruz
	fieldsToProcess := map[string]*string{
		"name":    &r.Name,
		"path":    &r.Path,
		"content": &r.Content,
		"image":   &r.Image,
		"target":  &r.Target,
		"mode":    &r.Mode,
		"owner":   &r.Owner,
		"group":   &r.Group,
		"command": &r.Command,
		"creates": &r.Creates,
		"only_if": &r.OnlyIf,
		"unless":  &r.Unless,
	}

	// Her bir alanı template motorundan geçiriyoruz
	for fieldName, val := range fieldsToProcess {
		if *val != "" {
			processed, err := config.ExecuteTemplate(*val, vars)
			if err != nil {
				return nil, fmt.Errorf("kaynak '%s' için '%s' alanı işlenirken şablon hatası: %w", r.Name, fieldName, err)
			}
			*val = processed
		}
	}

	id := r.Identify()
	switch r.Type {
	case "file":
		return &FileResource{
			CanonicalID: id, ResourceName: r.Name, Path: r.Path,
			Content: r.Content, Mode: r.Mode, Owner: r.Owner, Group: r.Group,
		}, nil
	case "exec":
		return &ExecResource{
			CanonicalID: id, Name: r.Name, Command: r.Command,
			Creates: r.Creates, OnlyIf: r.OnlyIf, Unless: r.Unless,
		}, nil
	case "package":
		return &PackageResource{CanonicalID: id, PackageName: r.Name, State: r.State, Provider: GetDefaultProvider()}, nil
	case "service":
		return &ServiceResource{CanonicalID: id, ServiceName: r.Name, DesiredState: r.State, Enabled: r.Enabled}, nil
	case "container":
		// HATA ÇÖZÜMÜ: map[string]string olan r.Env'yi []string ("KEY=VALUE") formatına çeviriyoruz.
		var envList []string
		for k, v := range r.Env {
			envList = append(envList, fmt.Sprintf("%s=%s", k, v))
		}

		return &ContainerResource{
			CanonicalID: id,
			Name:        r.Name,
			Image:       r.Image,
			State:       r.State,
			Ports:       r.Ports,
			Env:         envList, // Artık slice tipinde gönderiyoruz
			Volumes:     r.Volumes,
			Engine:      GetContainerEngine(),
		}, nil
	case "symlink":
		return &SymlinkResource{CanonicalID: id, Path: r.Path, Target: r.Target}, nil
	case "git":
		return &GitResource{CanonicalID: id, URL: r.URL, Path: r.Path}, nil
	default:
		return nil, fmt.Errorf("bilinmeyen kaynak tipi: %s", r.Type)
	}
}
