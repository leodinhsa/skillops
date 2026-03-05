package cmd

import (
	"fmt"
	"os"
	"skillops/internal/tui"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all downloaded skills",
	GroupID: "skill",
	Run: func(cmd *cobra.Command, args []string) {
		if err := tui.ShowList(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
