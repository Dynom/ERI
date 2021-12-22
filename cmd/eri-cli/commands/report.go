package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const (
	RFFull  ReportFormat = "full"
	RFStats ReportFormat = "stats"
)

const (
	outputFormatJSON = "json"
	outputFormatText = "text"
)

type ReportFormat string

type ReportSettings struct {
	OnlyInvalid bool
	Details     string
	Format      string
}

var reportSettings = &ReportSettings{}

type Encoder interface {
	Encode(v interface{}) error
}

type toEncode struct {
	fn func(v interface{}) error
}

func (t *toEncode) Encode(v interface{}) error {
	return t.fn(v)
}

// reportCmd represents the report command
var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Reporting companion to check",
	Long: `Some examples:
  - bzcat list.bz2 | eri-cli check | eri-cli report --only-invalid > report.json`,
	Args: func(cmd *cobra.Command, args []string) error {
		var stdInFromTerminal = term.IsTerminal(int(os.Stdin.Fd()))

		if len(args) > 0 {
			return errors.New("report doesn't take any arguments")
		}

		if stdInFromTerminal {
			return errors.New("report only reads from stdin")
		}

		switch reportSettings.Format {
		case outputFormatJSON:
		case outputFormatText:
		default:
			return fmt.Errorf("unsupported format %q", reportSettings.Format)
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		var encoder Encoder
		if reportSettings.Format == outputFormatJSON {
			encoder = json.NewEncoder(cmd.OutOrStdout())
		} else {
			encoder = &toEncode{
				fn: func(v interface{}) error {
					var err error
					if vt, ok := v.(CheckResultFull); ok {
						_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", vt.String())
					}

					return err
				},
			}
		}

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

	reportCmd.Flags().StringVar(&reportSettings.Format, "format", outputFormatJSON, "The format in which to report in, supported id: '"+outputFormatText+"' or '"+outputFormatJSON+"'")
	reportCmd.Flags().BoolVar(&reportSettings.OnlyInvalid, "only-invalid", false, "Only report rejected checks (ignored when report is stats)")
	reportCmd.Flags().StringVar(&reportSettings.Details, "details", string(RFFull), "Type of report, supported is: '"+string(RFStats)+"' or '"+string(RFFull)+"'")
}
