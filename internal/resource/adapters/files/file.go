package files

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/melih-ucgun/monarch/internal/core"
)

type FileAdapter struct {
	core.BaseResource
	Path       string
	Source     string // Kopyalanacak kaynak dosya (opsiyonel)
	Content    string // Yazılacak içerik (opsiyonel)
	Mode       os.FileMode
	State      string // present, absent
	BackupPath string // Yedeklenen dosyanın yolu
}

func NewFileAdapter(name string, params map[string]interface{}) *FileAdapter {
	path, _ := params["path"].(string)
	if path == "" {
		path = name // Eğer path verilmezse name'i path olarak kullan
	}

	source, _ := params["source"].(string)
	content, _ := params["content"].(string)
	state, _ := params["state"].(string)
	if state == "" {
		state = "present"
	}

	// İzinleri ayarla (varsayılan 0644)
	mode := os.FileMode(0644)
	if m, ok := params["mode"].(int); ok {
		mode = os.FileMode(m)
	}

	return &FileAdapter{
		BaseResource: core.BaseResource{Name: name, Type: "file"},
		Path:         path,
		Source:       source,
		Content:      content,
		Mode:         mode,
		State:        state,
	}
}

func (r *FileAdapter) Validate() error {
	if r.Path == "" {
		return fmt.Errorf("file path is required")
	}
	if r.State == "present" && r.Source == "" && r.Content == "" {
		return fmt.Errorf("either source or content must be provided for file resource")
	}
	return nil
}

func (r *FileAdapter) Check(ctx *core.SystemContext) (bool, error) {
	info, err := os.Stat(r.Path)

	if r.State == "absent" {
		// Dosya varsa silinmeli -> değişiklik var (true)
		return !os.IsNotExist(err), nil
	}

	// Dosya yoksa oluşturulmalı -> değişiklik var
	if os.IsNotExist(err) {
		return true, nil
	}
	if err != nil {
		return false, err
	}

	// İzin kontrolü
	if info.Mode().Perm() != r.Mode {
		return true, nil
	}

	// İçerik kontrolü
	if r.Content != "" {
		existingContent, err := os.ReadFile(r.Path)
		if err != nil {
			return false, err
		}
		if string(existingContent) != r.Content {
			return true, nil
		}
	} else if r.Source != "" {
		// Source ile hedefi karşılaştır
		same, err := compareFiles(r.Source, r.Path)
		if err != nil {
			return false, err
		}
		if !same {
			return true, nil
		}
	}

	return false, nil
}

func (r *FileAdapter) Apply(ctx *core.SystemContext) (core.Result, error) {
	needsAction, _ := r.Check(ctx)
	if !needsAction {
		return core.SuccessNoChange(fmt.Sprintf("File %s is up to date", r.Path)), nil
	}

	if ctx.DryRun {
		return core.SuccessChange(fmt.Sprintf("[DryRun] Would %s file %s", r.State, r.Path)), nil
	}

	// YEDEKLEME
	if core.GlobalBackup != nil {
		backupPath, err := core.GlobalBackup.BackupFile(r.Path)
		if err == nil {
			r.BackupPath = backupPath
		} else {
			return core.Failure(err, "Failed to backup file"), err
		}
	}

	if r.State == "absent" {
		if err := os.Remove(r.Path); err != nil {
			return core.Failure(err, "Failed to delete file"), err
		}
		return core.SuccessChange("File deleted"), nil
	}

	// Dizin yoksa oluştur
	dir := filepath.Dir(r.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return core.Failure(err, "Failed to create directory"), err
	}

	// İçerik yazma veya kopyalama
	if r.Content != "" {
		if err := os.WriteFile(r.Path, []byte(r.Content), r.Mode); err != nil {
			return core.Failure(err, "Failed to write content"), err
		}
	} else if r.Source != "" {
		if err := copyFile(r.Source, r.Path, r.Mode); err != nil {
			return core.Failure(err, "Failed to copy file"), err
		}
	}

	return core.SuccessChange(fmt.Sprintf("File %s created/updated", r.Path)), nil
}

func (r *FileAdapter) Revert(ctx *core.SystemContext) error {
	if r.BackupPath != "" {
		// Yedeği geri yükle
		return copyFile(r.BackupPath, r.Path, r.Mode)
	}

	if r.State == "present" {
		// Yedek yoksa ve dosya oluşturduysak, sil
		// (Dosya önceden yoktu demek)
		return os.Remove(r.Path)
	}

	return nil
}

// copyFile basit bir kopyalama fonksiyonu
func copyFile(src, dst string, mode os.FileMode) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}
	return os.Chmod(dst, mode)
}
