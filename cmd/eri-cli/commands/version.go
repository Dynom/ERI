package commands

import (
	"github.com/spf13/cobra"
)

var version string

func SetVersion(v string) {
	version = v
}

func init() {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "The version",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println(version)
		},
	}

	rootCmd.AddCommand(versionCmd)
}
