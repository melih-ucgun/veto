package config

import (
	"testing"
)

func TestSortResources(t *testing.T) {
	tests := []struct {
		name      string
		resources []Resource
		wantErr   bool
		expected  []string // Beklenen ID sırası
	}{
		{
			name: "Basit Doğrusal Bağımlılık",
			resources: []Resource{
				{Name: "app", DependsOn: []string{"pkg"}},
				{Name: "pkg", DependsOn: []string{}},
			},
			wantErr:  false,
			expected: []string{"pkg", "app"},
		},
		{
			name: "Karmaşık Bağımlılık Grafiği",
			resources: []Resource{
				{Name: "service", DependsOn: []string{"config"}},
				{Name: "config", DependsOn: []string{"pkg"}},
				{Name: "pkg", DependsOn: []string{}},
				{Name: "unrelated", DependsOn: []string{}},
			},
			wantErr:  false,
			expected: []string{"pkg", "unrelated", "config", "service"}, // pkg ve unrelated 0 derece ile başlar
		},
		{
			name: "Döngüsel Bağımlılık Hatası",
			resources: []Resource{
				{Name: "A", DependsOn: []string{"B"}},
				{Name: "B", DependsOn: []string{"A"}},
			},
			wantErr: true,
		},
		{
			name: "Eksik Bağımlılık Hatası",
			resources: []Resource{
				{Name: "A", DependsOn: []string{"olmayan_kaynak"}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SortResources(tt.resources)
			if (err != nil) != tt.wantErr {
				t.Errorf("SortResources() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if len(got) != len(tt.expected) {
				t.Errorf("Eksik veya fazla kaynak döndü. Got %d, Want %d", len(got), len(tt.expected))
				return
			}

			// Bağımlılık sırasını kontrol et (Basit check: sıralamada bağımlı olunan önce mi?)
			orderMap := make(map[string]int)
			for i, res := range got {
				orderMap[res.Name] = i
			}

			for _, res := range got {
				for _, dep := range res.DependsOn {
					if orderMap[dep] >= orderMap[res.Name] {
						t.Errorf("Bağımlılık ihlali: %s, %s'den önce gelmeliydi", dep, res.Name)
					}
				}
			}
		})
	}
}
