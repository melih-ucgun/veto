package files

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/melih-ucgun/monarch/internal/core"
)

type TemplateAdapter struct {
	core.BaseResource
	Src        string                 // Template dosyasının yolu
	Dest       string                 // Çıktı dosyasının yolu
	Vars       map[string]interface{} // Template içine gönderilecek değişkenler
	Mode       os.FileMode
	BackupPath string
}

func NewTemplateAdapter(name string, params map[string]interface{}) *TemplateAdapter {
	src, _ := params["src"].(string)
	dest, _ := params["dest"].(string)
	if dest == "" {
		dest = name
	}

	vars := make(map[string]interface{})
	if v, ok := params["vars"].(map[string]interface{}); ok {
		vars = v
	}

	mode := os.FileMode(0644)
	if m, ok := params["mode"].(int); ok {
		mode = os.FileMode(m)
	}

	return &TemplateAdapter{
		BaseResource: core.BaseResource{Name: name, Type: "template"},
		Src:          src,
		Dest:         dest,
		Vars:         vars,
		Mode:         mode,
	}
}

func (r *TemplateAdapter) Validate() error {
	if r.Src == "" || r.Dest == "" {
		return fmt.Errorf("template requires 'src' and 'dest'")
	}
	return nil
}

func (r *TemplateAdapter) Check(ctx *core.SystemContext) (bool, error) {
	// 1. Hedef dosya yoksa -> Değişiklik var
	if _, err := os.Stat(r.Dest); os.IsNotExist(err) {
		return true, nil
	}

	// 2. İçerik kontrolü: Template'i bellekte render et ve mevcut dosyayla karşılaştır
	rendered, err := r.render()
	if err != nil {
		return false, fmt.Errorf("failed to render template for check: %w", err)
	}

	currentContent, err := os.ReadFile(r.Dest)
	if err != nil {
		return false, err
	}

	if string(currentContent) != rendered {
		return true, nil
	}

	return false, nil
}

func (r *TemplateAdapter) Apply(ctx *core.SystemContext) (core.Result, error) {
	needsAction, err := r.Check(ctx)
	if err != nil {
		return core.Failure(err, "Template check failed"), err
	}
	if !needsAction {
		return core.SuccessNoChange(fmt.Sprintf("Template %s is up to date", r.Dest)), nil
	}

	if ctx.DryRun {
		return core.SuccessChange(fmt.Sprintf("[DryRun] Render template %s to %s", r.Src, r.Dest)), nil
	}

	// YEDEKLEME
	if core.GlobalBackup != nil {
		backupPath, err := core.GlobalBackup.BackupFile(r.Dest)
		if err == nil {
			r.BackupPath = backupPath
		}
	}

	// Render et
	content, err := r.render()
	if err != nil {
		return core.Failure(err, "Template render failed"), err
	}

	// Klasörü oluştur
	if err := os.MkdirAll(filepath.Dir(r.Dest), 0755); err != nil {
		return core.Failure(err, "Failed to create directory"), err
	}

	// Yaz
	if err := os.WriteFile(r.Dest, []byte(content), r.Mode); err != nil {
		return core.Failure(err, "Failed to write file"), err
	}

	return core.SuccessChange("Template applied successfully"), nil
}

func (r *TemplateAdapter) Revert(ctx *core.SystemContext) error {
	if r.BackupPath != "" {
		return copyFile(r.BackupPath, r.Dest, r.Mode)
	}
	return os.Remove(r.Dest)
}

func (r *TemplateAdapter) render() (string, error) {
	// Template dosyasını oku
	tmplContent, err := os.ReadFile(r.Src)
	if err != nil {
		return "", err
	}

	// Parse et
	t, err := template.New(filepath.Base(r.Src)).Parse(string(tmplContent))
	if err != nil {
		return "", err
	}

	// Execute et (Buffer'a yaz)
	var buf bytes.Buffer
	if err := t.Execute(&buf, r.Vars); err != nil {
		return "", err
	}

	return buf.String(), nil
}
