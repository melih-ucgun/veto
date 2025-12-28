package resources

import (
	"fmt"

	"github.com/melih-ucgun/monarch/internal/config"
)

func New(r config.Resource, vars map[string]interface{}) (Resource, error) {
	processedContent := r.Content
	if r.Content != "" {
		var err error
		processedContent, err = config.ExecuteTemplate(r.Content, vars)
		if err != nil {
			return nil, fmt.Errorf("[%s] şablon işleme hatası: %w", r.Name, err)
		}
	}

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

	case "git":
		return &GitResource{
			URL:  r.URL,
			Path: r.Path,
		}, nil

	case "exec":
		return &ExecResource{
			Name:    r.Name,
			Command: r.Content, // Komutu 'content' alanından alıyoruz
		}, nil

	case "noop":
		return nil, nil

	default:
		return nil, fmt.Errorf("bilinmeyen kaynak tipi: %s", r.Type)
	}
}
