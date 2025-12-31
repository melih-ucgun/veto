package core

import (
	"fmt"
	"sync"
	"time"

	"github.com/pterm/pterm"

	"github.com/melih-ucgun/veto/internal/config"
	"github.com/melih-ucgun/veto/internal/state"
)

// StateUpdater interface allows Engine to be independent of the state package.
type StateUpdater interface {
	UpdateResource(resType, name, targetState, status string) error
}

// ConfigItem is the raw configuration part that the engine will process.
type ConfigItem struct {
	Name   string
	Type   string
	State  string
	When   string // Condition to evaluate
	Params map[string]interface{}
}

// Engine is the main structure managing resources.
type Engine struct {
	Context        *SystemContext
	StateUpdater   StateUpdater // Optional: State manager
	AppliedHistory []ApplyableResource
}

// NewEngine creates a new engine instance.
func NewEngine(ctx *SystemContext, updater StateUpdater) *Engine {
	// Initialize Backup Manager
	_ = InitBackupManager() // Ignore error for now (or log)
	return &Engine{
		Context:      ctx,
		StateUpdater: updater,
	}
}

// ResourceCreator fonksiyon tipi
type ResourceCreator func(resType, name string, params map[string]interface{}, ctx *SystemContext) (ApplyableResource, error)

// ApplyableResource interface
type ApplyableResource interface {
	Apply(ctx *SystemContext) (Result, error)
	GetName() string
	GetType() string
}

// Run processes the given configuration list.
func (e *Engine) Run(items []ConfigItem, createFn ResourceCreator) error {
	errCount := 0

	// Transaction recording
	transaction := state.Transaction{
		ID:        state.GenerateID(),
		Timestamp: time.Now().Format(time.RFC3339),
		Status:    "success",
		Changes:   []state.TransactionChange{},
	}

	for _, item := range items {
		// Params preparation
		if item.Params == nil {
			item.Params = make(map[string]interface{})
		}
		item.Params["state"] = item.State

		// 1. Create resource
		res, err := createFn(item.Type, item.Name, item.Params, e.Context)
		if err != nil {
			Failure(err, "Skipping invalid resource definition: "+item.Name)
			errCount++
			continue
		}

		// 2. Apply resource
		result, err := res.Apply(e.Context)

		status := "success"
		if err != nil {
			status = "failed"
			errCount++
			fmt.Printf("❌ [%s] Failed: %v\n", item.Name, err)
		} else if result.Changed {
			fmt.Printf("✅ [%s] %s\n", item.Name, result.Message)

			// Record change for History
			change := state.TransactionChange{
				Type:   item.Type,
				Name:   item.Name,
				Action: "applied", // Could be more specific based on result message
			}

			// Try to get target path (specifically for file)
			if p, ok := item.Params["path"].(string); ok {
				change.Target = p
			} else {
				change.Target = item.Name // Fallback
			}

			// Use local interface to avoid import cycle
			type Backupable interface {
				GetBackupPath() string
			}

			if b, ok := res.(Backupable); ok {
				change.BackupPath = b.GetBackupPath()
			}

			transaction.Changes = append(transaction.Changes, change)

		} else {
			msg := "OK"
			if result.Message != "" {
				msg = result.Message
			}
			pterm.Info.Printf("[%s] %s: %s\n", item.Type, item.Name, msg)
		}

		// 3. Save State (If not DryRun)
		if !e.Context.DryRun && e.StateUpdater != nil {
			// Save as "failed" even if it failed, to track the attempt
			saveErr := e.StateUpdater.UpdateResource(item.Type, item.Name, item.State, status)
			if saveErr != nil {
				fmt.Printf("⚠️ Warning: Failed to save state for %s: %v\n", item.Name, saveErr)
			}
		}
	}

	if errCount > 0 {
		transaction.Status = "failed"
	}

	// Save History
	if !e.Context.DryRun {
		hm := state.NewHistoryManager("")
		if err := hm.AddTransaction(transaction); err != nil {
			fmt.Printf("⚠️ Warning: Failed to save history: %v\n", err)
		}
	}

	if errCount > 0 {
		return fmt.Errorf("encountered %d errors during execution", errCount)
	}
	return nil
}

// RunParallel processes configuration items in the given layer in parallel.
func (e *Engine) RunParallel(layer []ConfigItem, createFn ResourceCreator) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(layer))
	var updatedResources []ApplyableResource // Track successful ones (For Rollback)
	var mu sync.Mutex                        // lock for updatedResources

	// Transaction recording
	transaction := state.Transaction{
		ID:        state.GenerateID(),
		Timestamp: time.Now().Format(time.RFC3339),
		Status:    "success",
		Changes:   []state.TransactionChange{},
	}
	var txMu sync.Mutex

	for _, item := range layer {
		wg.Add(1)
		go func(it ConfigItem) {
			defer wg.Done()

			// Params preparation
			if it.Params == nil {
				it.Params = make(map[string]interface{})
			}
			it.Params["state"] = it.State

			// 0. Check Condition (When)
			if it.When != "" {
				shouldRun, err := EvaluateCondition(it.When, e.Context)
				if err != nil {
					pterm.Error.Printf("[%s] Condition Error: %v\n", it.Name, err)
					errChan <- err
					return
				}
				if !shouldRun {
					// pterm.Gray is a color, to print we need to creating a style or use Printf from a style
					pterm.NewStyle(pterm.FgGray).Printf("⚪ [%s] Skipped (Condition not met: %s)\n", it.Name, it.When)
					return
				}
			}

			// 0.5 Render Templates in Params
			if err := renderParams(it.Params, e.Context); err != nil {
				pterm.Error.Printf("[%s] Template Error: %v\n", it.Name, err)
				errChan <- err
				return
			}

			// 1. Create resource
			res, err := createFn(it.Type, it.Name, it.Params, e.Context)
			if err != nil {
				Failure(err, "Skipping invalid resource definition: "+it.Name)
				errChan <- err
				return
			}

			// 2. Apply resource
			result, err := res.Apply(e.Context)

			status := "success"

			if err != nil {
				status = "failed"
				errChan <- err
				pterm.Error.Printf("[%s] %s: Failed: %v\n", it.Type, it.Name, err)
			} else if result.Changed {
				// Success
				pterm.Success.Printf("[%s] %s: %s\n", it.Type, it.Name, result.Message)

				// Save successful changes (For Rollback)
				if !e.Context.DryRun {
					mu.Lock()
					updatedResources = append(updatedResources, res)
					mu.Unlock()
				}

				// Record change for History
				change := state.TransactionChange{
					Type:   it.Type,
					Name:   it.Name,
					Action: "applied",
				}

				// Try to get target path
				if p, ok := it.Params["path"].(string); ok {
					change.Target = p
				} else {
					change.Target = it.Name // Fallback
				}

				// Use local interface to avoid import cycle
				type Backupable interface {
					GetBackupPath() string
				}

				if b, ok := res.(Backupable); ok {
					change.BackupPath = b.GetBackupPath()
				}

				txMu.Lock()
				transaction.Changes = append(transaction.Changes, change)
				txMu.Unlock()

			} else {
				// No Change (Info or Skipped)
				msg := "OK"
				if result.Message != "" {
					msg = result.Message
				}
				pterm.Info.Printf("[%s] %s: %s\n", it.Type, it.Name, msg)
			}

			// 3. Save State
			if !e.Context.DryRun && e.StateUpdater != nil {
				e.StateUpdater.UpdateResource(it.Type, it.Name, it.State, status)
			}
		}(item)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	errCount := 0
	for range errChan {
		errCount++
	}

	if errCount > 0 {
		transaction.Status = "failed"
		// Trigger Rollback
		if !e.Context.DryRun {
			pterm.Println()
			pterm.Error.Println("Error occurred. Initiating Rollback...")

			// 1. First revert operations that succeeded in the current layer (but incomplete due to other errors)
			pterm.Warning.Printf("Visualizing Rollback for current layer (%d resources)...\n", len(updatedResources))
			e.rollback(updatedResources)

			// 2. Revert operations completed in previous layers
			pterm.Warning.Printf("Visualizing Rollback for previous layers (%d resources)...\n", len(e.AppliedHistory))
			e.rollback(e.AppliedHistory)

			transaction.Status = "reverted"
		}
	}

	// Save History
	if !e.Context.DryRun {
		hm := state.NewHistoryManager("")
		if err := hm.AddTransaction(transaction); err != nil {
			fmt.Printf("⚠️ Warning: Failed to save history: %v\n", err)
		}
	}

	if errCount > 0 {
		return fmt.Errorf("encountered %d errors in parallel layer execution", errCount)
	}

	// Add successful ones to global history
	// Note: Must be LIFO for Revert order. rollback function iterates specifically in reverse.
	// We add FIFO to AppliedHistory (append).
	// Example: Layer0 (A, B) -> AppliedHistory=[A, B]
	// Layer1 (C, D) -> Fail. CurrentRevert(C). HistoryRevert(A, B) -> B revert, A revert. Correct.
	e.AppliedHistory = append(e.AppliedHistory, updatedResources...)

	return nil
}

// rollback reverts the given list of resources in reverse order.
func (e *Engine) rollback(resources []ApplyableResource) {
	// Go in reverse order
	for i := len(resources) - 1; i >= 0; i-- {
		res := resources[i]
		if rev, ok := res.(Revertable); ok {
			pterm.Warning.Printf("Visualizing Rollback for %s...\n", res.GetName())
			if err := rev.Revert(e.Context); err != nil {
				pterm.Error.Printf("Failed to revert %s: %v\n", res.GetName(), err)
				if !e.Context.DryRun && e.StateUpdater != nil {
					_ = e.StateUpdater.UpdateResource(res.GetType(), res.GetName(), "any", "revert_failed")
				}
			} else {
				pterm.Success.Printf("Reverted %s\n", res.GetName())
				if !e.Context.DryRun && e.StateUpdater != nil {
					// Successful revert, mark as 'reverted'
					_ = e.StateUpdater.UpdateResource(res.GetType(), res.GetName(), "any", "reverted")
				}
			}
		}
	}
}

// renderParams traverses the map and renders any string values as templates.
func renderParams(params map[string]interface{}, ctx *SystemContext) error {
	for k, v := range params {
		switch val := v.(type) {
		case string:
			rendered, err := config.ExecuteTemplate(val, ctx)
			if err != nil {
				return fmt.Errorf("param '%s': %w", k, err)
			}
			params[k] = rendered
		case map[string]interface{}:
			// Recursive
			if err := renderParams(val, ctx); err != nil {
				return err
			}
		case []interface{}:
			// Iterate slice
			for i, item := range val {
				if str, ok := item.(string); ok {
					rendered, err := config.ExecuteTemplate(str, ctx)
					if err != nil {
						return fmt.Errorf("param '%s' index %d: %w", k, i, err)
					}
					val[i] = rendered
				} else if subMap, ok := item.(map[string]interface{}); ok {
					if err := renderParams(subMap, ctx); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}
