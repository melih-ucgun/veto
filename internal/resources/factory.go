package resources

import (
	"fmt"

	"github.com/melih-ucgun/monarch/internal/config"
)

func New(r config.Resource, vars map[string]interface{}) (Resource, error) {
	// Şablonlama işlemi (name, path, content, mode vb. için)
	fieldsToProcess := map[string]*string{
		"name":    &r.Name,
		"path":    &r.Path,
		"content": &r.Content,
		"image":   &r.Image,
		"target":  &r.Target,
		"mode":    &r.Mode,
		"owner":   &r.Owner,
		"group":   &r.Group,
	}

	for _, val := range fieldsToProcess {
		if *val != "" {
			processed, err := config.ExecuteTemplate(*val, vars)
			if err == nil {
				*val = processed
			}
		}
	}

	canonicalID := r.Identify()

	switch r.Type {
	case "file":
		return &FileResource{
			CanonicalID:  canonicalID,
			ResourceName: r.Name,
			Path:         r.Path,
			Content:      r.Content,
			Mode:         r.Mode,
			Owner:        r.Owner,
			Group:        r.Group,
		}, nil
	// ... diğer caseler aynı kalacak
	case "package":
		return &PackageResource{CanonicalID: canonicalID, PackageName: r.Name, State: r.State, Provider: GetDefaultProvider()}, nil
	case "service":
		return &ServiceResource{CanonicalID: canonicalID, ServiceName: r.Name, DesiredState: r.State, Enabled: r.Enabled}, nil
	case "container":
		return &ContainerResource{
			CanonicalID: canonicalID,
			Name:        r.Name,
			Image:       r.Image,
			State:       r.State,
			Ports:       r.Ports,
			Env:         r.Env,
			Volumes:     r.Volumes,
			Engine:      GetContainerEngine(),
		}, nil
	case "symlink":
		return &SymlinkResource{
			CanonicalID: canonicalID,
			Path:        r.Path,
			Target:      r.Target,
		}, nil
	case "git":
		return &GitResource{CanonicalID: canonicalID, URL: r.URL, Path: r.Path}, nil
	case "exec":
		return &ExecResource{CanonicalID: canonicalID, Name: r.Name, Command: r.Command}, nil
	default:
		return nil, fmt.Errorf("bilinmeyen kaynak tipi: %s", r.Type)
	}
}
