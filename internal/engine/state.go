package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type ResourceState struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	LastApplied time.Time `json:"last_applied"`
	Success     bool      `json:"success"`
}

type State struct {
	Resources map[string]ResourceState `json:"resources"`
}

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

// Merge, farklı makinelerden gelen state bilgilerini ana state ile birleştirir.
func (s *State) Merge(other *State) {
	if other == nil || other.Resources == nil {
		return
	}
	if s.Resources == nil {
		s.Resources = make(map[string]ResourceState)
	}
	for id, resState := range other.Resources {
		s.Resources[id] = resState
	}
}

func getStatePath() string {
	return ".monarch/state.json"
}
