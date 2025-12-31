package config

import (
	"reflect"
	"testing"
)

func TestSortResources(t *testing.T) {
	tests := []struct {
		name      string
		resources []ResourceConfig
		expected  [][]string // Sadece ID'leri kontrol edeceğiz
		wantErr   bool
	}{
		{
			name:      "Empty input",
			resources: []ResourceConfig{},
			expected:  nil,
			wantErr:   false,
		},
		{
			name: "Linear dependency (A -> B -> C)",
			resources: []ResourceConfig{
				{ID: "C", DependsOn: []string{"B"}},
				{ID: "A"}, // Independent
				{ID: "B", DependsOn: []string{"A"}},
			},
			// Layer 0: A
			// Layer 1: B
			// Layer 2: C
			expected: [][]string{{"A"}, {"B"}, {"C"}},
			wantErr:  false,
		},
		{
			name: "Branching (A -> B, A -> C)",
			resources: []ResourceConfig{
				{ID: "B", DependsOn: []string{"A"}},
				{ID: "C", DependsOn: []string{"A"}},
				{ID: "A"},
			},
			// Layer 0: A
			// Layer 1: B, C (order within layer is implementation dependent, but both in L1)
			// For this test we assume deterministic order if queue is sorted,
			// but our implementation appends in queue order. Map iteration is random.
			// Let's loosen the check or just check layers count and containment.
			// To make test stable, the implementation would need to sort keys.
			// Current implementation does NOT sort keys, so B and C can be swapped.
			// We will check manually in test logic.
			expected: [][]string{{"A"}, {"B", "C"}},
			wantErr:  false,
		},
		{
			name: "Cycle check (A -> B -> A)",
			resources: []ResourceConfig{
				{ID: "A", DependsOn: []string{"B"}},
				{ID: "B", DependsOn: []string{"A"}},
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name: "Self dependency (A -> A)",
			resources: []ResourceConfig{
				{ID: "A", DependsOn: []string{"A"}},
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name: "Missing dependency",
			resources: []ResourceConfig{
				{ID: "A", DependsOn: []string{"Z"}},
			},
			expected: nil,
			wantErr:  true,
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

			// Convert got to [][]string IDs for comparison
			var gotIDs [][]string
			if got != nil {
				gotIDs = make([][]string, len(got))
				for i, layer := range got {
					for _, r := range layer {
						gotIDs[i] = append(gotIDs[i], r.ID)
					}
				}
			}

			// Special check for branching test (order agnostic in layer)
			if tt.name == "Branching (A -> B, A -> C)" {
				if len(gotIDs) != 2 {
					t.Errorf("Expected 2 layers, got %d", len(gotIDs))
				}
				if gotIDs[0][0] != "A" {
					t.Errorf("Layer 0 should be A")
				}
				// Layer 1 should contain B and C
				layer1 := gotIDs[1]
				if len(layer1) != 2 {
					t.Errorf("Layer 1 should have 2 items")
				}
				hasB := layer1[0] == "B" || layer1[1] == "B"
				hasC := layer1[0] == "C" || layer1[1] == "C"
				if !hasB || !hasC {
					t.Errorf("Layer 1 should contain B and C, got %v", layer1)
				}
				return
			}

			if !reflect.DeepEqual(gotIDs, tt.expected) {
				t.Errorf("SortResources() = %v, want %v", gotIDs, tt.expected)
			}
		})
	}
}
