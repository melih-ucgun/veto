package cmd

import (
	"fmt"
	"time"

	"github.com/melih-ucgun/monarch/internal/state"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "View application history",
	Run: func(cmd *cobra.Command, args []string) {
		hm := state.NewHistoryManager("")
		history, err := hm.LoadHistory()
		if err != nil {
			pterm.Error.Println("Failed to load history:", err)
			return
		}

		if len(history) == 0 {
			pterm.Info.Println("No history found.")
			return
		}

		pterm.DefaultHeader.Println("Transaction History")

		tableData := [][]string{{"ID", "Date", "Status", "Changes"}}

		// Show latest first (reverse iteration)
		for i := len(history) - 1; i >= 0; i-- {
			tx := history[i]
			t, _ := time.Parse(time.RFC3339, tx.Timestamp)
			dateStr := t.Format("2006-01-02 15:04:05")

			statusStyle := pterm.NewStyle(pterm.FgGreen)
			if tx.Status == "failed" {
				statusStyle = pterm.NewStyle(pterm.FgRed)
			} else if tx.Status == "reverted" {
				statusStyle = pterm.NewStyle(pterm.FgYellow)
			}

			tableData = append(tableData, []string{
				tx.ID,
				dateStr,
				statusStyle.Sprint(tx.Status),
				fmt.Sprintf("%d", len(tx.Changes)),
			})
		}

		pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
	},
}

func init() {
	rootCmd.AddCommand(historyCmd)
}
