package core

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// MockTransport implements core.Transport
type MockTransport struct {
	CapturedCmds []string
}

func (m *MockTransport) Execute(ctx context.Context, cmd string) (string, error) {
	m.CapturedCmds = append(m.CapturedCmds, cmd)
	return "ok", nil
}
func (m *MockTransport) CopyFile(ctx context.Context, src, dst string) error     { return nil }
func (m *MockTransport) DownloadFile(ctx context.Context, src, dst string) error { return nil }
func (m *MockTransport) GetFileSystem() FileSystem                               { return &RealFS{} }
func (m *MockTransport) GetOS(ctx context.Context) (string, error)               { return "linux", nil }
func (m *MockTransport) Close() error                                            { return nil }

// MockResource implements Resource and Revertable
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

func (m *MockResource) Check(ctx *SystemContext) (bool, error) {
	return true, nil // Always true for tests unless specified
}

func (m *MockResource) Validate(ctx *SystemContext) error {
	return nil
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
	ctx := NewSystemContext(false, nil)

	t.Run("All success", func(t *testing.T) {
		engine := NewEngine(ctx, nil)

		res1 := &MockResource{Name: "res1", ApplyResult: SuccessChange("ok")}
		res2 := &MockResource{Name: "res2", ApplyResult: SuccessNoChange("ok")}

		// Mock Creator function
		createFn := func(t, n string, p map[string]interface{}, c *SystemContext) (Resource, error) {
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

		createFn := func(t, n string, p map[string]interface{}, c *SystemContext) (Resource, error) {
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

		createFn := func(t, n string, p map[string]interface{}, c *SystemContext) (Resource, error) {
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

	t.Run("Hooks execution", func(t *testing.T) {
		mockTransport := &MockTransport{}
		ctx := NewSystemContext(false, mockTransport)
		engine := NewEngine(ctx, nil)

		res := &MockResource{Name: "resHook", ApplyResult: SuccessChange("ok")}

		createFn := func(t, n string, p map[string]interface{}, c *SystemContext) (Resource, error) {
			return res, nil
		}

		item := ConfigItem{
			Name: "resHook",
			Hooks: Hooks{
				Pre:      "echo pre",
				Post:     "echo post",
				OnChange: "echo change",
			},
		}

		err := engine.RunParallel([]ConfigItem{item}, createFn)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Verify executed commands
		expected := []string{"echo pre", "echo post", "echo change"}
		if len(mockTransport.CapturedCmds) != 3 {
			t.Fatalf("Expected 3 hooks, got %d: %v", len(mockTransport.CapturedCmds), mockTransport.CapturedCmds)
		}
		for i, exp := range expected {
			if mockTransport.CapturedCmds[i] != exp {
				t.Errorf("Hook %d mismatch: want %s, got %s", i, exp, mockTransport.CapturedCmds[i])
			}
		}
	})
}
