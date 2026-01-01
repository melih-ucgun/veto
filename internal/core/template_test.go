package core

import (
	"testing"
)

func TestExecuteTemplate(t *testing.T) {
	tests := []struct {
		name    string
		content string
		vars    map[string]interface{}
		want    string
		wantErr bool
	}{
		{
			name:    "Basit Değişken Değişimi",
			content: "Merhaba {{ .user }}!",
			vars:    map[string]interface{}{"user": "melih"},
			want:    "Merhaba melih!",
			wantErr: false,
		},
		{
			name:    "Sayısal Değerler",
			content: "Port: {{ .port }}",
			vars:    map[string]interface{}{"port": 8080},
			want:    "Port: 8080",
			wantErr: false,
		},
		{
			name:    "Hatalı Template Sözdizimi",
			content: "Merhaba {{ .user",
			vars:    map[string]interface{}{"user": "melih"},
			want:    "",
			wantErr: true,
		},
		{
			name:    "Eksik Değişken (Default Davranış)",
			content: "Değer: {{ .yok }}",
			vars:    map[string]interface{}{},
			want:    "",
			wantErr: true, // Implementation uses Option("missingkey=error")
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExecuteTemplate(tt.content, tt.vars)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want && !tt.wantErr {
				t.Errorf("ExecuteTemplate() got = %v, want %v", got, tt.want)
			}
		})
	}
}
