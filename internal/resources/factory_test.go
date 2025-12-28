package resources

import (
	"testing"

	"github.com/melih-ucgun/monarch/internal/config"
)

func TestNewResourceFactory(t *testing.T) {
	vars := map[string]interface{}{"user": "melih"}

	tests := []struct {
		name    string
		resCfg  config.Resource
		wantErr bool
	}{
		{
			name: "Dosya Kaynağı Oluşturma",
			resCfg: config.Resource{
				Type:    "file",
				Name:    "test-file",
				Path:    "/tmp/test",
				Content: "hello {{ .user }}",
			},
			wantErr: false,
		},
		{
			name: "Paket Kaynağı Oluşturma",
			resCfg: config.Resource{
				Type:  "package",
				Name:  "neovim",
				State: "installed",
			},
			wantErr: false,
		},
		{
			name: "Bilinmeyen Kaynak Tipi",
			resCfg: config.Resource{
				Type: "unknown_type",
				Name: "fail",
			},
			wantErr: true,
		},
		{
			name: "Hatalı Şablon İçeriği",
			resCfg: config.Resource{
				Type:    "file",
				Name:    "bad-template",
				Content: "{{ .user",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := New(tt.resCfg, vars)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && res == nil {
				t.Error("New() kaynak oluşturamadı (nil döndü)")
			}

			// Tip kontrolü (File için içerik işlenmiş mi?)
			if !tt.wantErr && tt.resCfg.Type == "file" {
				fr := res.(*FileResource)
				if fr.Content != "hello melih" {
					t.Errorf("Şablon doğru işlenmedi. Got: %s", fr.Content)
				}
			}
		})
	}
}
