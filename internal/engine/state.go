package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// ResourceState, tek bir kaynağın durum bilgisini tutar.
type ResourceState struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	LastApplied time.Time `json:"last_applied"`
	Success     bool      `json:"success"`
}

// State, tüm sistemin Monarch tarafından bilinen durumudur.
type State struct {
	Resources map[string]ResourceState `json:"resources"`
}

// LoadState, yerel state.json dosyasını yükler.
func LoadState() (*State, error) {
	path := getStatePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &State{Resources: make(map[string]ResourceState)}, nil
		}
		return nil, err
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// Save, mevcut durumu dosyaya kaydeder.
func (s *State) Save() error {
	path := getStatePath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// UpdateResource, bir kaynağın durumunu günceller.
func (s *State) UpdateResource(id, resType string, success bool) {
	if s.Resources == nil {
		s.Resources = make(map[string]ResourceState)
	}
	s.Resources[id] = ResourceState{
		ID:          id,
		Type:        resType,
		LastApplied: time.Now(),
		Success:     success,
	}
}

func getStatePath() string {
	// Geliştirme kolaylığı için yerel dizinde tutuyoruz.
	// İleride /var/lib/monarch/state.json gibi bir konuma taşınabilir.
	return ".monarch/state.json"
}
