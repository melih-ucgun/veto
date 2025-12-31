package discovery

import (
	"os"
	"path/filepath"
	"strings"
)

// CommonConfigMap maps package/tool names to their standard config paths.
// Paths starting with "~" will be expanded using userHome.
var CommonConfigMap = map[string][]string{
	// Shells
	"zsh":  {"~/.zshrc", "~/.zshenv", "~/.zprofile"},
	"bash": {"~/.bashrc", "~/.bash_profile", "~/.bash_aliases"},
	"fish": {"~/.config/fish/config.fish"},

	// Editors
	"vim":    {"~/.vimrc"},
	"neovim": {"~/.config/nvim/init.lua", "~/.config/nvim/init.vim"},
	"nvim":   {"~/.config/nvim/init.lua", "~/.config/nvim/init.vim"}, // Alias
	"nano":   {"~/.nanorc"},
	"emacs":  {"~/.emacs", "~/.emacs.d/init.el"},
	"vscode": {"~/.config/Code/User/settings.json", "~/.config/Code/User/keybindings.json"},
	"code":   {"~/.config/Code/User/settings.json", "~/.config/Code/User/keybindings.json"},

	// Tools
	"git":       {"~/.gitconfig"},
	"ssh":       {"~/.ssh/config"},
	"tmux":      {"~/.tmux.conf"},
	"alacritty": {"~/.config/alacritty/alacritty.yml", "~/.config/alacritty/alacritty.toml"},
	"kitty":     {"~/.config/kitty/kitty.conf"},
	"starship":  {"~/.config/starship.toml"},
	"gh":        {"~/.config/gh/config.yml"},
	"i3":        {"~/.config/i3/config"},
	"sway":      {"~/.config/sway/config"},
	"hyprland":  {"~/.config/hypr/hyprland.conf"},
	"waybar":    {"~/.config/waybar/config", "~/.config/waybar/style.css"},
	"rofi":      {"~/.config/rofi/config.rasi"},
	"wofi":      {"~/.config/wofi/config"},

	// Services (System-wide)
	"nginx":    {"/etc/nginx/nginx.conf"},
	"docker":   {"/etc/docker/daemon.json"},
	"samba":    {"/etc/samba/smb.conf"},
	"sshd":     {"/etc/ssh/sshd_config"},
	"postgres": {"/var/lib/postgres/data/postgresql.conf"}, // Varies heavily
}

// DiscoverConfigs suggests config files based on selected packages.
func DiscoverConfigs(packages []string, userHome string) ([]string, error) {
	var foundConfigs []string

	for _, pkg := range packages {
		// Handle exact match or partial match logic if needed
		// For now, exact match on map key
		if paths, ok := CommonConfigMap[strings.ToLower(pkg)]; ok {
			for _, p := range paths {
				absPath := expandPath(p, userHome)
				if fileExists(absPath) {
					foundConfigs = append(foundConfigs, absPath)
				}
			}
		}
	}

	return unique(foundConfigs), nil
}

func expandPath(path, home string) string {
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	return path
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
