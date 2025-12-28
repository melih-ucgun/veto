package resources

import (
	"bytes"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"
)

type FileResource struct {
	CanonicalID  string
	ResourceName string
	Path         string
	Content      string
	Mode         string
	Owner        string
	Group        string
}

func (f *FileResource) ID() string {
	return f.CanonicalID
}

// Check, hem içeriği hem de meta verileri kontrol eder.
func (f *FileResource) Check() (bool, error) {
	info, err := os.Stat(f.Path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	// 1. İçerik Kontrolü
	currentContent, err := os.ReadFile(f.Path)
	if err != nil {
		return false, err
	}
	if !bytes.Equal(currentContent, []byte(f.Content)) {
		return false, nil
	}

	// 2. İzin (Mode) Kontrolü
	if f.Mode != "" {
		targetMode, _ := strconv.ParseUint(f.Mode, 8, 32)
		if uint32(info.Mode().Perm()) != uint32(targetMode) {
			return false, nil
		}
	}

	// 3. Sahiplik Kontrolü
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		if f.Owner != "" {
			targetUID, _ := resolveUser(f.Owner)
			if stat.Uid != uint32(targetUID) {
				return false, nil
			}
		}
		if f.Group != "" {
			targetGID, _ := resolveGroup(f.Group)
			if stat.Gid != uint32(targetGID) {
				return false, nil
			}
		}
	}

	return true, nil
}

func (f *FileResource) Diff() (string, error) {
	info, err := os.Stat(f.Path)
	if os.IsNotExist(err) {
		return fmt.Sprintf("+ file: %s (Yeni oluşturulacak)", f.Path), nil
	}

	diffMsg := ""
	currentContent, _ := os.ReadFile(f.Path)
	if string(currentContent) != f.Content {
		diffMsg += "~ İçerik değişecek\n"
	}

	if f.Mode != "" {
		targetMode, _ := strconv.ParseUint(f.Mode, 8, 32)
		if uint32(info.Mode().Perm()) != uint32(targetMode) {
			diffMsg += fmt.Sprintf("~ İzinler: %o -> %s\n", info.Mode().Perm(), f.Mode)
		}
	}

	if f.Owner != "" || f.Group != "" {
		diffMsg += "~ Sahiplik/Grup değişebilir\n"
	}

	if diffMsg == "" {
		return "", nil
	}
	return fmt.Sprintf("! %s:\n%s", f.Path, diffMsg), nil
}

func (f *FileResource) Apply() error {
	dir := filepath.Dir(f.Path)
	os.MkdirAll(dir, 0o755)

	// Dosyayı yaz
	err := os.WriteFile(f.Path, []byte(f.Content), 0o644)
	if err != nil {
		return err
	}

	// İzinleri uygula
	if f.Mode != "" {
		m, _ := strconv.ParseUint(f.Mode, 8, 32)
		os.Chmod(f.Path, os.FileMode(m))
	}

	// Sahipliği uygula
	if f.Owner != "" || f.Group != "" {
		uid, _ := resolveUser(f.Owner)
		gid, _ := resolveGroup(f.Group)
		// -1 değeri değişikliği atla demektir
		os.Chown(f.Path, uid, gid)
	}

	return nil
}

// Yardımcı Fonksiyonlar: Kullanıcı adı -> UID, Grup adı -> GID
func resolveUser(name string) (int, error) {
	if name == "" {
		return -1, nil
	}
	u, err := user.Lookup(name)
	if err != nil {
		// Eğer sayısal bir değerse doğrudan onu kullanmayı dene
		if id, errID := strconv.Atoi(name); errID == nil {
			return id, nil
		}
		return -1, err
	}
	return strconv.Atoi(u.Uid)
}

func resolveGroup(name string) (int, error) {
	if name == "" {
		return -1, nil
	}
	g, err := user.LookupGroup(name)
	if err != nil {
		if id, errID := strconv.Atoi(name); errID == nil {
			return id, nil
		}
		return -1, err
	}
	return strconv.Atoi(g.Gid)
}
