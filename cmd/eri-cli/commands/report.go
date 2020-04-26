package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

type ReportSettings struct {
	OnlyInvalid bool
	Details     string
}

var reportSettings = &ReportSettings{}

// reportCmd represents the report command
var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("report called, args: %#v\n", args)
		fmt.Printf("cmd: %+v\n", cmd)

		cmd.InOrStdin()
	},
}

func init() {
	rootCmd.AddCommand(reportCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// reportCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	reportCmd.Flags().BoolVar(&reportSettings.OnlyInvalid, "only-invalid", false, "Only report rejected checks")
	reportCmd.Flags().StringVar(&reportSettings.Details, "details", "full", "Type of report")
}
