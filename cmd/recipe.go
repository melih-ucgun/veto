package cmd

import (
	"github.com/melih-ucgun/monarch/internal/hub"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var recipeCmd = &cobra.Command{
	Use:   "recipe",
	Short: "Manage configuration recipes",
	Long:  `Manage multiple Monarch configuration recipes (e.g., work, personal). Each recipe contains a system.yaml and a rulesets directory.`,
}

var recipeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all recipes",
	Run: func(cmd *cobra.Command, args []string) {
		mgr := hub.NewRecipeManager("")
		recipes, err := mgr.List()
		if err != nil {
			pterm.Error.Println("Failed to list recipes:", err)
			return
		}

		active, _ := mgr.GetActive()

		pterm.DefaultHeader.Println("Available Recipes")
		if len(recipes) == 0 {
			pterm.Info.Println("No recipes found. Create one with 'monarch recipe create <name>'")
			return
		}

		tableData := [][]string{{"Name", "Status"}}

		for _, p := range recipes {
			status := ""
			if p == active {
				status = "Active"
			}
			tableData = append(tableData, []string{p, status})
		}
		pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
	},
}

var recipeCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new recipe",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		mgr := hub.NewRecipeManager("")

		if err := mgr.Create(name); err != nil {
			pterm.Error.Println("Failed to create recipe:", err)
			return
		}
		pterm.Success.Printf("Recipe '%s' created successfully.\n", name)
		pterm.Info.Printf("Recipe path: %s/recipes/%s\n", mgr.BaseDir, name)
		pterm.Info.Println("Use 'monarch recipe use <name>' to activate it.")
	},
}

var recipeUseCmd = &cobra.Command{
	Use:   "use [name]",
	Short: "Switch to a recipe",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		mgr := hub.NewRecipeManager("")

		if err := mgr.Use(name); err != nil {
			pterm.Error.Println("Failed to switch recipe:", err)
			return
		}
		pterm.Success.Printf("Switched to recipe '%s'.\n", name)
	},
}

var recipeShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show active recipe",
	Run: func(cmd *cobra.Command, args []string) {
		mgr := hub.NewRecipeManager("")
		active, err := mgr.GetActive()
		if err != nil {
			pterm.Error.Println(err)
			return
		}
		if active == "" {
			pterm.Warning.Println("No active recipe set.")
		} else {
			pterm.Info.Printf("Active Recipe: %s\n", active)
		}
	},
}

func init() {
	rootCmd.AddCommand(recipeCmd)
	recipeCmd.AddCommand(recipeListCmd)
	recipeCmd.AddCommand(recipeCreateCmd)
	recipeCmd.AddCommand(recipeUseCmd)
	recipeCmd.AddCommand(recipeShowCmd)
}
