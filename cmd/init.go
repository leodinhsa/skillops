package cmd

import (
	"fmt"
	"os"
	"skillops/internal/tui"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:     "init",
	Short:   "Declare which IDE tools are active in this project",
	GroupID: "project",
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if err := tui.RunInit(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
