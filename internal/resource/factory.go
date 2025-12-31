package resource

import (
	"fmt"

	"github.com/melih-ucgun/monarch/internal/core"

	"github.com/melih-ucgun/monarch/internal/resource/adapters/files"
	"github.com/melih-ucgun/monarch/internal/resource/adapters/identity"
	"github.com/melih-ucgun/monarch/internal/resource/adapters/pkgmngs"
	"github.com/melih-ucgun/monarch/internal/resource/adapters/scm"
	"github.com/melih-ucgun/monarch/internal/resource/adapters/service"
	"github.com/melih-ucgun/monarch/internal/resource/adapters/shell"
)

// Deprecated fonksiyon placeholder
func CreateResource(resType string, name string, state string, ctx *core.SystemContext) (core.Resource, error) {
	return nil, fmt.Errorf("use CreateResourceWithParams")
}

// CreateResourceWithParams artık core.Resource döndürüyor
func CreateResourceWithParams(resType string, name string, params map[string]interface{}, ctx *core.SystemContext) (core.Resource, error) {

	stateParam, _ := params["state"].(string)
	if stateParam == "" {
		stateParam = "present"
	}

	switch resType {
	// Package Managers
	case "pacman":
		return pkgmngs.NewPacmanAdapter(name, stateParam), nil
	case "apt":
		return pkgmngs.NewAptAdapter(name, stateParam), nil
	case "dnf":
		return pkgmngs.NewDnfAdapter(name, stateParam), nil
	case "brew":
		return pkgmngs.NewBrewAdapter(name, stateParam), nil
	case "apk":
		return pkgmngs.NewApkAdapter(name, stateParam), nil
	case "flatpak":
		return pkgmngs.NewFlatpakAdapter(name, stateParam), nil
	case "snap":
		return pkgmngs.NewSnapAdapter(name, stateParam), nil
	case "zypper":
		return pkgmngs.NewZypperAdapter(name, stateParam), nil
	case "yum":
		return pkgmngs.NewYumAdapter(name, stateParam), nil
	case "paru":
		return pkgmngs.NewParuAdapter(name, stateParam), nil
	case "yay":
		return pkgmngs.NewYayAdapter(name, stateParam), nil
	case "package", "pkg":
		return detectPackageManager(name, stateParam, ctx)

	// Filesystem
	case "file":
		params["state"] = stateParam
		return files.NewFileAdapter(name, params), nil
	case "symlink":
		params["state"] = stateParam
		return files.NewSymlinkAdapter(name, params), nil
	case "archive", "extract":
		return files.NewArchiveAdapter(name, params), nil
	case "download":
		return files.NewDownloadAdapter(name, params), nil
	case "template":
		return files.NewTemplateAdapter(name, params), nil
	case "line_in_file", "lineinfile":
		params["state"] = stateParam
		return files.NewLineInFileAdapter(name, params), nil

	// Identity
	case "user":
		params["state"] = stateParam
		return identity.NewUserAdapter(name, params), nil
	case "group":
		params["state"] = stateParam
		return identity.NewGroupAdapter(name, params), nil

	// Others
	case "git":
		params["state"] = stateParam
		return scm.NewGitAdapter(name, params), nil
	case "service", "systemd":
		params["state"] = stateParam
		return service.NewServiceAdapter(name, params), nil
	case "exec", "shell", "cmd":
		return shell.NewExecAdapter(name, params), nil

	default:
		return nil, fmt.Errorf("unknown resource type: %s", resType)
	}
}

func detectPackageManager(name, state string, ctx *core.SystemContext) (core.Resource, error) {
	switch ctx.Distro {
	case "arch", "cachyos", "manjaro", "endeavouros":
		return pkgmngs.NewPacmanAdapter(name, state), nil
	case "ubuntu", "debian", "pop", "mint", "kali":
		return pkgmngs.NewAptAdapter(name, state), nil
	case "fedora", "rhel", "centos", "almalinux":
		return pkgmngs.NewDnfAdapter(name, state), nil
	case "alpine":
		return pkgmngs.NewApkAdapter(name, state), nil
	case "opensuse", "sles":
		return pkgmngs.NewZypperAdapter(name, state), nil
	case "darwin":
		return pkgmngs.NewBrewAdapter(name, state), nil
	default:
		return nil, fmt.Errorf("automatic package manager detection failed")
	}
}
