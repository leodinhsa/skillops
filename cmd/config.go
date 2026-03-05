package cmd

import (
	"fmt"
	"os"
	"skillops/internal/config"
	"skillops/internal/tui"
	"skillops/internal/utils"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:     "config",
	GroupID: "agentic",
	Short:   "Manage global Agentic IDE configurations",
}

var addAgenticCmd = &cobra.Command{
	Use:   "add-agentic",
	Short: "Register a new IDE type",
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		path, _ := cmd.Flags().GetString("path")

		if name == "" || path == "" {
			fmt.Println("Error: name and path are required. Use -n and -p flags.")
			return
		}

		if err := utils.ValidateName(name); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if err := config.AddAgentic(name, path); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("🚀 %s\n", tui.SuccessStyle.Render(fmt.Sprintf("Added agentic '%s' with path '%s'", name, path)))
	},
}

var removeAgenticCmd = &cobra.Command{
	Use:   "remove-agentic",
	Short: "Remove a registered IDE mapping",
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			fmt.Println("Error: name is required. Use -n flag.")
			return
		}

		if err := config.RemoveAgentic(name); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("🗑️  %s\n", tui.SuccessStyle.Render(fmt.Sprintf("Removed agentic '%s'", name)))
	},
}

var updateAgenticCmd = &cobra.Command{
	Use:   "update-agentic",
	Short: "Update an existing IDE mapping",
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		path, _ := cmd.Flags().GetString("path")

		if name == "" || path == "" {
			fmt.Println("Error: name and path are required. Use -n and -p flags.")
			return
		}

		if err := config.UpdateAgentic(name, path); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("🔄 %s\n", tui.SuccessStyle.Render(fmt.Sprintf("Updated agentic '%s' to path '%s'", name, path)))
	},
}

func init() {
	addAgenticCmd.Flags().StringP("name", "n", "", "Name of the agentic IDE")
	addAgenticCmd.Flags().StringP("path", "p", "", "Relative path to skills folder")

	removeAgenticCmd.Flags().StringP("name", "n", "", "Name of the agentic IDE")

	updateAgenticCmd.Flags().StringP("name", "n", "", "Name of the agentic IDE")
	updateAgenticCmd.Flags().StringP("path", "p", "", "New relative path")

	configCmd.AddCommand(addAgenticCmd)
	configCmd.AddCommand(removeAgenticCmd)
	configCmd.AddCommand(updateAgenticCmd)
	rootCmd.AddCommand(configCmd)
}
