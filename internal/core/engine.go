package core

import (
	"fmt"
	"sync"
	"time"

	"github.com/pterm/pterm"

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
	AppliedHistory []Resource
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
type ResourceCreator func(resType, name string, params map[string]interface{}, ctx *SystemContext) (Resource, error)

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
	var updatedResources []Resource // Track successful ones (For Rollback)
	var mu sync.Mutex               // lock for updatedResources

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
	e.AppliedHistory = append(e.AppliedHistory, updatedResources...)

	return nil
}

// PlanResult represents the outcome of a Plan operation.
type PlanResult struct {
	Changes []PlanChange
}

// PlanChange represents a single proposed change.
type PlanChange struct {
	Type   string
	Name   string
	Action string // "create", "modify", "noop"
	Diff   string // Optional detailed diff
}

// Plan generates a preview of changes without applying them.
func (e *Engine) Plan(items []ConfigItem, createFn ResourceCreator) (*PlanResult, error) {
	result := &PlanResult{
		Changes: []PlanChange{},
	}

	for _, item := range items {
		// Params preparation
		if item.Params == nil {
			item.Params = make(map[string]interface{})
		}
		item.Params["state"] = item.State

		// 0. Check Condition (When)
		if item.When != "" {
			shouldRun, err := EvaluateCondition(item.When, e.Context)
			if err != nil {
				return nil, fmt.Errorf("[%s] Condition Error: %w", item.Name, err)
			}
			if !shouldRun {
				continue // Skip silently or add as "skipped"
			}
		}

		// 0.5 Render Templates
		if err := renderParams(item.Params, e.Context); err != nil {
			return nil, fmt.Errorf("[%s] Template Error: %w", item.Name, err)
		}

		// 1. Create resource
		resApp, err := createFn(item.Type, item.Name, item.Params, e.Context)
		if err != nil {
			return nil, fmt.Errorf("[%s] Creation Error: %w", item.Name, err)
		}

		// 2. Check State
		var action string
		var diff string

		if checker, ok := resApp.(interface {
			Check(ctx *SystemContext) (bool, error)
		}); ok {
			needsAction, err := checker.Check(e.Context)
			if err != nil {
				return nil, fmt.Errorf("[%s] Check Error: %w", item.Name, err)
			}

			if needsAction {
				action = "apply"
			} else {
				action = "noop"
			}
		} else {
			action = "unknown"
		}

		if action != "noop" {
			result.Changes = append(result.Changes, PlanChange{
				Type:   item.Type,
				Name:   item.Name,
				Action: action,
				Diff:   diff,
			})
		}
	}

	return result, nil
}

// rollback reverts the given list of resources in reverse order.
func (e *Engine) rollback(resources []Resource) {
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
			rendered, err := ExecuteTemplate(val, ctx)
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
					rendered, err := ExecuteTemplate(str, ctx)
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

// Prune identifies unmanaged resources and removes them upon confirmation.
// Supports any resource type that implements the Lister interface.
func (e *Engine) Prune(configItems []ConfigItem, createFn ResourceCreator) error {
	// 1. Identify Resource Types present in config or supported for pruning
	targetTypes := []string{"pkg", "service"}

	pterm.Println()
	pterm.DefaultHeader.WithFullWidth().
		WithBackgroundStyle(pterm.NewStyle(pterm.BgRed)).
		WithTextStyle(pterm.NewStyle(pterm.FgWhite, pterm.Bold)).
		Println("PRUNE MODE (Destructive)")

	totalUnmanaged := 0
	type PruneTask struct {
		Type      string
		Resources []string
		Adapter   Lister
	}
	var tasks []PruneTask

	for _, resType := range targetTypes {
		// Collect Managed Resources for this type
		managed := make(map[string]bool)
		for _, item := range configItems {
			if item.Type == resType {
				managed[item.Name] = true
			}
		}

		// Create dummy adapter for listing
		// Use a fixed name "prune_helper" to get an instance
		dummyRes, err := createFn(resType, "prune_helper", nil, e.Context)
		if err != nil {
			// Resource type might not be registered or supported on this OS
			continue
		}

		lister, ok := dummyRes.(Lister)
		if !ok {
			// Type does not support listing, skip
			continue
		}

		// List Installed
		pterm.Info.Printf("Analyzing %s resources...\n", resType)
		installed, err := lister.ListInstalled(e.Context)
		if err != nil {
			pterm.Warning.Printf("Failed to list installed %s: %v\n", resType, err)
			continue
		}

		// Calculate Diff
		var unmanaged []string
		for _, name := range installed {
			if !managed[name] {
				unmanaged = append(unmanaged, name)
			}
		}

		if len(unmanaged) > 0 {
			tasks = append(tasks, PruneTask{
				Type:      resType,
				Resources: unmanaged,
				Adapter:   lister,
			})
			totalUnmanaged += len(unmanaged)

			pterm.Error.Printf("Found %d unmanaged %s resources:\n", len(unmanaged), resType)
			for i, name := range unmanaged {
				if i < 5 {
					fmt.Printf(" - %s\n", name)
				} else {
					fmt.Printf(" ... and %d more\n", len(unmanaged)-5)
					break
				}
			}
		}
	}

	if totalUnmanaged == 0 {
		pterm.Success.Println("System is clean! No unmanaged resources found.")
		return nil
	}

	// Double Confirmation
	pterm.Println()
	result, _ := pterm.DefaultInteractiveConfirm.
		WithDefaultText(fmt.Sprintf("Are you sure you want to delete/disable these %d resources?", totalUnmanaged)).
		WithDefaultValue(false).
		Show()

	if !result {
		pterm.Info.Println("Prune cancelled.")
		return nil
	}

	pterm.Println()
	pterm.Warning.Println("This operation is DESTRUCTIVE.")
	input, _ := pterm.DefaultInteractiveTextInput.
		WithDefaultText("Type 'confirm prune' to proceed").
		Show()

	if input != "confirm prune" {
		pterm.Error.Println("Confirmation failed. Aborting.")
		return nil
	}

	// Execution
	pterm.Println()
	pterm.Info.Println("Starting cleanup...")

	for _, task := range tasks {
		for _, name := range task.Resources {
			pterm.Printf("Pruning [%s] %s... ", task.Type, name)

			// Create resource with state=absent (for pkg) or appropriate state for services
			params := make(map[string]interface{})
			if task.Type == "service" {
				params["enabled"] = false
				params["state"] = "stopped"
			} else {
				params["state"] = "absent"
			}

			res, err := createFn(task.Type, name, params, e.Context)
			if err != nil {
				pterm.Error.Printf("Failed to create resource handle: %v\n", err)
				continue
			}

			if e.Context.DryRun {
				pterm.Success.Println("[DryRun] Pruned")
				continue
			}

			_, err = res.Apply(e.Context)
			if err != nil {
				pterm.Error.Printf("Failed: %v\n", err)
			} else {
				pterm.Success.Println("Pruned")
			}
		}
	}

	return nil
}
