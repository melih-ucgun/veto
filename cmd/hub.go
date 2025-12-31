package cmd

import (
	"os"
	"path/filepath"

	"github.com/melih-ucgun/monarch/internal/hub"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var hubCmd = &cobra.Command{
	Use:   "hub",
	Short: "Interact with Monarch Hub",
	Long:  `Search, install, and manage RuleSets from Monarch Hub.`,
}

var hubSearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search for rulesets",
	Run: func(cmd *cobra.Command, args []string) {
		query := ""
		if len(args) > 0 {
			query = args[0]
		}
		client := hub.NewHubClient()
		results, err := client.Search(query)
		if err != nil {
			pterm.Error.Println("Search failed:", err)
			return
		}

		pterm.DefaultHeader.Printf("Hub Search Results: '%s'\n", query)
		if len(results) == 0 {
			pterm.Info.Println("No rulesets found.")
			return
		}

		tableData := [][]string{{"Name"}}
		for _, r := range results {
			tableData = append(tableData, []string{r})
		}
		pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
	},
}

var hubInstallCmd = &cobra.Command{
	Use:   "install [ruleset]",
	Short: "Install a ruleset to the active recipe",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		rulesetName := args[0]

		mgr := hub.NewRecipeManager("")
		activeDir, err := mgr.GetActiveRecipeDir()
		if err != nil {
			pterm.Error.Println("Failed to get active recipe:", err)
			return
		}

		rulesetsDir := filepath.Join(activeDir, "rulesets")

		pterm.Info.Printf("Installing '%s' to %s...\n", rulesetName, rulesetsDir)

		client := hub.NewHubClient()
		if err := client.Install(rulesetName, rulesetsDir); err != nil {
			pterm.Error.Println("Failed to install ruleset:", err)
			return
		}

		pterm.Success.Printf("RuleSet '%s' installed successfully.\n", rulesetName)
		pterm.Info.Println("Tip: Add the ruleset to your system.yaml to enable it.")
	},
}

var hubRemoveCmd = &cobra.Command{
	Use:   "remove [ruleset]",
	Short: "Remove a ruleset from the active recipe",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		rulesetName := args[0]

		mgr := hub.NewRecipeManager("")
		activeDir, err := mgr.GetActiveRecipeDir()
		if err != nil {
			pterm.Error.Println("Failed to get active recipe:", err)
			return
		}

		targetDir := filepath.Join(activeDir, "rulesets", rulesetName)

		// Safety check
		if _, err := os.Stat(targetDir); os.IsNotExist(err) {
			pterm.Error.Printf("RuleSet '%s' not found.\n", rulesetName)
			return
		}

		if err := os.RemoveAll(targetDir); err != nil {
			pterm.Error.Println("Failed to remove ruleset:", err)
			return
		}

		pterm.Success.Printf("RuleSet '%s' removed.\n", rulesetName)
	},
}

func init() {
	rootCmd.AddCommand(hubCmd)
	hubCmd.AddCommand(hubSearchCmd)
	hubCmd.AddCommand(hubInstallCmd)
	hubCmd.AddCommand(hubRemoveCmd)
}
