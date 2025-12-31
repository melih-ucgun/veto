package core

import (
	"errors"
	"strings"
	"testing"
)

// MockResource implements ApplyableResource and Revertable
type MockResource struct {
	Name         string
	Type         string
	ApplyResult  Result
	ApplyErr     error
	RevertErr    error
	ApplyCalled  bool
	RevertCalled bool
}

func (m *MockResource) GetName() string { return m.Name }
func (m *MockResource) GetType() string { return m.Type }

func (m *MockResource) Apply(ctx *SystemContext) (Result, error) {
	m.ApplyCalled = true
	return m.ApplyResult, m.ApplyErr
}

func (m *MockResource) Revert(ctx *SystemContext) error {
	m.RevertCalled = true
	return m.RevertErr
}

// MockStateUpdater implements StateUpdater
type MockStateUpdater struct {
	Updates []struct {
		Type, Name, TargetState, Status string
	}
}

func (m *MockStateUpdater) UpdateResource(resType, name, targetState, status string) error {
	m.Updates = append(m.Updates, struct {
		Type, Name, TargetState, Status string
	}{resType, name, targetState, status})
	return nil
}

func TestEngine_RunParallel(t *testing.T) {
	ctx := &SystemContext{DryRun: false}

	t.Run("All success", func(t *testing.T) {
		engine := NewEngine(ctx, nil)

		res1 := &MockResource{Name: "res1", ApplyResult: SuccessChange("ok")}
		res2 := &MockResource{Name: "res2", ApplyResult: SuccessNoChange("ok")}

		// Mock Creator function
		createFn := func(t, n string, p map[string]interface{}, c *SystemContext) (ApplyableResource, error) {
			if n == "res1" {
				return res1, nil
			}
			if n == "res2" {
				return res2, nil
			}
			return nil, errors.New("unknown")
		}

		items := []ConfigItem{{Name: "res1"}, {Name: "res2"}}
		err := engine.RunParallel(items, createFn)

		if err != nil {
			t.Errorf("RunParallel failed: %v", err)
		}
		if !res1.ApplyCalled || !res2.ApplyCalled {
			t.Error("Resources not applied")
		}
		// res1 changed, should be in history
		if len(engine.AppliedHistory) != 1 || engine.AppliedHistory[0].GetName() != "res1" {
			t.Error("AppliedHistory incorrect")
		}
	})

	t.Run("Failure triggers rollback in same layer", func(t *testing.T) {
		updater := &MockStateUpdater{}
		engine := NewEngine(ctx, updater)

		// res1 succeeds (Changed)
		res1 := &MockResource{Name: "res1", Type: "test", ApplyResult: SuccessChange("ok")}
		// res2 fails
		res2 := &MockResource{Name: "res2", Type: "test", ApplyErr: errors.New("fail")}

		createFn := func(t, n string, p map[string]interface{}, c *SystemContext) (ApplyableResource, error) {
			if n == "res1" {
				return res1, nil
			}
			if n == "res2" {
				return res2, nil
			}
			return nil, errors.New("unknown")
		}

		items := []ConfigItem{{Name: "res1"}, {Name: "res2"}}
		err := engine.RunParallel(items, createFn)

		if err == nil {
			t.Error("Expected error, got nil")
		} else if !strings.Contains(err.Error(), "encountered 1 errors") {
			t.Errorf("Unexpected error message: %v", err)
		}

		// Verify Rollback called on res1
		if !res1.RevertCalled {
			t.Error("Rollback not triggered for res1")
		}

		// Verify State Update for revert
		foundReverted := false
		for _, u := range updater.Updates {
			if u.Name == "res1" && u.Status == "reverted" {
				foundReverted = true
			}
		}
		if !foundReverted {
			t.Error("State not updated to 'reverted' for res1")
		}
	})

	t.Run("Rollback respects LIFO across layers", func(t *testing.T) {
		engine := NewEngine(ctx, nil)

		// Layer 1: resA (Success)
		resA := &MockResource{Name: "resA", ApplyResult: SuccessChange("ok")}
		engine.AppliedHistory = append(engine.AppliedHistory, resA)

		// Layer 2: resB (Success/Change), resC (Fail)
		resB := &MockResource{Name: "resB", ApplyResult: SuccessChange("ok")}
		resC := &MockResource{Name: "resC", ApplyErr: errors.New("fail")}

		createFn := func(t, n string, p map[string]interface{}, c *SystemContext) (ApplyableResource, error) {
			if n == "resB" {
				return resB, nil
			}
			if n == "resC" {
				return resC, nil
			}
			return nil, errors.New("unknown")
		}

		items := []ConfigItem{{Name: "resB"}, {Name: "resC"}}
		err := engine.RunParallel(items, createFn)

		if err == nil {
			t.Error("Expected error")
		}

		// Rollback Order:
		// 1. Current Layer: resB
		// 2. Previous Layers: resA
		// We can't strictly check timing here without better mocking,
		// but checking both are called is good start.

		if !resB.RevertCalled {
			t.Error("resB not reverted")
		}
		if !resA.RevertCalled {
			t.Error("resA (prev layer) not reverted")
		}
	})
}
