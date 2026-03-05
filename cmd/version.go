package cmd

import (
	"fmt"
	"skillops/internal/config"
	"skillops/internal/tui"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the current version of skillops",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println()
		fmt.Print(tui.TitleStyle.Render(" SKILLOPS VERSION "))
		fmt.Printf("\nCurrent Version: %s\n", tui.SuccessStyle.Render(config.Version))
		fmt.Println()
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
