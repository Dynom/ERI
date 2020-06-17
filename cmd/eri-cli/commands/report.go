package commands

import (
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/spf13/cobra"
)

const (
	RFFull  ReportFormat = "full"
	RFStats ReportFormat = "stats"
)

type ReportFormat string

type ReportSettings struct {
	OnlyInvalid bool
	Details     string
}

var reportSettings = &ReportSettings{}

// reportCmd represents the report command
var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Reporting companion to check",
	Long: `Some examples:
  - bzcat list.bz2 | eri-cli check | eri-cli report --only-invalid > report.json`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errors.New("report doesn't take any arguments")
		}

		if !isStdinPiped() {
			return errors.New("report only reads from stdin")
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		encoder := json.NewEncoder(cmd.OutOrStdout())

		decoder := json.NewDecoder(cmd.InOrStdin())
		decoder.DisallowUnknownFields()

		if reportSettings.Details == string(RFStats) {
			now := time.Now()
			var report = ReportStats{}
			for {
				var cr CheckResultFull
				err := decoder.Decode(&cr)
				if err == io.EOF {
					break
				}

				if err != nil {
					cmd.PrintErrf("Error trying to read report %s\n", err)
					continue
				}

				if cr.Valid {
					report.Passed++
				} else {
					report.Rejected++
				}
			}

			report.Duration = time.Since(now).Milliseconds()
			err := encoder.Encode(report)
			if err != nil {
				cmd.PrintErrf("Error trying to write report %s\n", err)
			}

			return
		}

		for {
			var cr CheckResultFull
			err := decoder.Decode(&cr)
			if err == io.EOF {
				break
			}

			if err != nil {
				cmd.PrintErrf("Error trying to read report %s\n", err)
				continue
			}

			if reportSettings.OnlyInvalid && cr.Valid {
				continue
			}

			err = encoder.Encode(cr)
			if err != nil {
				cmd.PrintErrf("Error trying to write report %s\n", err)
				break
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(reportCmd)

	reportCmd.Flags().BoolVar(&reportSettings.OnlyInvalid, "only-invalid", false, "Only report rejected checks (ignored when report is stats)")
	reportCmd.Flags().StringVar(&reportSettings.Details, "details", "full", "Type of report, supported is: 'stats' or 'full'")
}
