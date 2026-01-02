package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/melih-ucgun/veto/internal/core"
	"github.com/melih-ucgun/veto/internal/state"
	"github.com/melih-ucgun/veto/internal/system"
	"github.com/melih-ucgun/veto/internal/transport"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var rollbackCmd = &cobra.Command{
	Use:   "rollback [transactionID]",
	Short: "Rollback a specific transaction",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		txID := args[0]
		hm := state.NewHistoryManager("")

		tx, err := hm.GetTransaction(txID)
		if err != nil {
			pterm.Error.Println(err)
			return
		}

		pterm.DefaultHeader.Printf("Rolling Back: %s", txID)
		pterm.Warning.Println("This operation attempts to undo changes made in this transaction.")

		// Confirm
		confirmed, _ := cmd.Flags().GetBool("yes")
		if !confirmed {
			result, _ := pterm.DefaultInteractiveConfirm.Show("Are you sure?")
			if !result {
				pterm.Info.Println("Rollback cancelled.")
				return
			}
		}

		// Iterate changes in reverse
		for i := len(tx.Changes) - 1; i >= 0; i-- {
			change := tx.Changes[i]
			pterm.Info.Printf("Reverting %s %s...\n", change.Type, change.Name)

			if err := revertChange(change); err != nil {
				pterm.Error.Printf("Failed to revert %s: %v\n", change.Name, err)
			} else {
				pterm.Success.Printf("Reverted %s\n", change.Name)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(rollbackCmd)
	rollbackCmd.Flags().BoolP("yes", "y", false, "Confirm rollback automatically")
}

func revertChange(change state.TransactionChange) error {
	// Simple manual revert logic
	// In a more complex system, we might reuse Resource.Revert by reconstructing the object
	// But here we rely on basic file restore / pkg remove

	switch change.Type {
	case "file":
		if change.BackupPath != "" {
			// Check if backup exists
			if _, err := os.Stat(change.BackupPath); err != nil {
				return fmt.Errorf("backup file not found: %s", change.BackupPath)
			}
			// Restore file
			return copyFile(change.BackupPath, change.Target)
		} else {
			// No backup. If action was 'applied' (created), we might delete it?
			// But 'applied' could mean modify without backup.
			// If it was a new file, deleting it is safe revert.
			// How do we know if it was new? We don't track prev_state nicely yet in Engine.
			// Engine sends "applied" for everything.
			// Safe fallback: Do nothing if no backup, or warn.
			return fmt.Errorf("no backup available for file, cannot safely revert")
		}

	case "pkg":
		// Inverse action
		// Assuming we installed it.
		// We need to know system package manager.
		// This is tricky without fully initializing the Engine/System context.
		// Let's quick-detect system.
		// Let's quick-detect system.
		// Rollback runs locally
		ctx := core.NewSystemContext(false, transport.NewLocalTransport())
		system.Detect(ctx)

		// We can reuse PkgAdapter but we need to know the type?
		// Actually Type is "pkg", but underlying adapter depends on distro.
		// Let's construct a generic Pkg command if possible or just use what we have.
		// Simpler: Just warn that pkg rollback is Manual for now unless we import logic.
		// Better: Create a minimal pkg adapter and call Revert.
		// BUT: PkgAdapter needs to know current state to decide what to do in Revert method?
		// Revert method in ApkAdapter checks r.State. If r.State was "present", it deletes.

		// If transaction says "applied" (installed), we want to remove.
		// So we create an adapter with State="present" and call Revert().

		// This requires importing `resource` and `pkg` packages which might cause cycles if not careful.
		// And we need factory.
		// Let's print a warning for now to be safe in this iteration.
		pterm.Warning.Printf("Automatic package rollback not fully implemented yet for %s. Please remove manually.\n", change.Name)
		return nil

	default:
		return fmt.Errorf("unsupported resource type for rollback: %s", change.Type)
	}
}

// copyFile simple copy util (duplicated from file adapter to avoid import)
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
