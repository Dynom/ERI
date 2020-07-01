package commands

import (
	"context"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "eri-cli",
	Short: "CLI Compagnion of ERI",
	Long:  ``,
}

func Execute() {
	if err := rootCmd.ExecuteContext(context.Background()); err != nil {
		os.Exit(1)
	}
}
