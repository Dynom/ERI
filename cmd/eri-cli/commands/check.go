package commands

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"github.com/Dynom/ERI/types"
	"github.com/Dynom/ERI/validator"
	"github.com/Dynom/ERI/validator/validations"
	scsv "github.com/smartystreets/scanners/csv"
	"github.com/spf13/cobra"
)

type CheckResultFull struct {
	Email  string   `json:"email"`
	Passed bool     `json:"passed"`
	Checks []string `json:"check_run"`
}

type CheckSettings struct {
	Format string
	CSV    csvOptions
	Check  checkOptions
}

type checkOptions struct {
	Resolver net.IP
}

type csvOptions struct {
	skipRows uint64
	column   uint64
}

var (
	checkSettings = &CheckSettings{}
)

// checkCmd represents the check command
var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Validate email addresses",
	Long:  ``,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return errors.New("too many arguments, expected 0 or 1")
		}

		if len(args) == 0 && !isStdinPiped() {
			return errors.New("missing argument")
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		var dialer = &net.Dialer{}
		if checkSettings.Check.Resolver != nil {
			setCustomResolver(dialer, checkSettings.Check.Resolver)
		}

		v := validator.NewEmailAddressValidator(dialer)

		var it *ScanIterator
		if len(args) > 0 {
			it = createTextIterator(strings.NewReader(args[0]))
		} else if isStdinPiped() {
			switch checkSettings.Format {
			case "":
				fallthrough
			case "text":
				it = createTextIterator(os.Stdin)
			case "csv":
				it = createCSVIterator(os.Stdin)
			default:
				cmd.PrintErrf("bad format %q", checkSettings.Format)
				return
			}
		}

		if it == nil {
			cmd.PrintErr("No suitable iterator found, this is.. unexpected.")
			return
		}

		for it.Next() {
			email, err := it.Value()
			if err != nil {
				cmd.PrintErr(err)
				continue
			}

			cmd.Printf("Value to check %q\n", email)
			r, err := check(cmd.Context(), v.CheckWithLookup, email)
			if err != nil {
				cmd.PrintErr(err)
				continue
			}

			cmd.Printf("Result: %+v\n", r)
		}
	},
}

func setCustomResolver(dialer *net.Dialer, ip net.IP) {
	if dialer == nil {
		dialer = &net.Dialer{}
	}

	if dialer.Resolver == nil {
		dialer.Resolver = &net.Resolver{
			PreferGo: true,
		}
	}

	dialer.Resolver.Dial = func(ctx context.Context, network, address string) (conn net.Conn, e error) {
		d := net.Dialer{}
		return d.DialContext(ctx, network, net.JoinHostPort(ip.String(), `53`))
	}
}

func check(ctx context.Context, fn validator.CheckFn, email string) (CheckResultFull, error) {
	parts, err := types.NewEmailParts(email)
	if err != nil {
		return CheckResultFull{}, err
	}

	var result = CheckResultFull{Email: email}

	checkResult := fn(ctx, parts)
	fmt.Printf("CR: %+v\n", checkResult)
	{
		copy := checkResult.Validations
		flags := validations.Flag(copy.RemoveFlag(validations.FValid))
		result.Checks = flags.AsStringSlice()
		result.Passed = checkResult.Validations.IsValid()
	}

	return result, nil
}

func createTextIterator(r io.Reader) *ScanIterator {
	scanner := bufio.NewScanner(r)

	return NewScanIterator(
		scanner.Scan,
		func() (string, error) {
			return scanner.Text(), nil
		},
		func() error {
			return nil
		},
	)
}

func createCSVIterator(r io.Reader) *ScanIterator {
	scanner := scsv.NewScanner(
		r,
		scsv.SkipRecords(int(checkSettings.CSV.skipRows)),
		scsv.ReuseRecord(true),
		scsv.FieldsPerRecord(int(checkSettings.CSV.column)),
	)

	return NewScanIterator(
		scanner.Scan,
		func() (string, error) {
			record := scanner.Record()
			if uint64(len(record)-1) < checkSettings.CSV.column {
				return "", fmt.Errorf("column index %d exceeds columns in row, skipping row", checkSettings.CSV.column)
			}
			return record[checkSettings.CSV.column], nil
		},
		scanner.Error,
	)
}

func NewScanIterator(next func() bool, value func() (string, error), close func() error) *ScanIterator {
	return &ScanIterator{
		next:  next,
		value: value,
		close: close,
	}
}

type ScanIterator struct {
	next  func() bool
	value func() (string, error)
	close func() error
}

func (i *ScanIterator) Next() bool {
	return i.next()
}

func (i *ScanIterator) Value() (string, error) {
	return i.value()
}

func (i *ScanIterator) Close() error {
	return i.close()
}

func init() {
	rootCmd.AddCommand(checkCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// checkCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// checkCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	//checkCmd.Flags().StringVar(&checkSettings.From, "from", "", "")
	checkCmd.Flags().StringVar(&checkSettings.Format, "format", "text", "text or csv. Text means a single email address per line '\\n'")
	checkCmd.Flags().Uint64Var(&checkSettings.CSV.skipRows, "csv-skip-rows", 0, "Rows to skip, useful when wanting to skip the header in CSV files")
	checkCmd.Flags().Uint64Var(&checkSettings.CSV.column, "csv-column", 0, "The column to read email addresses from, 0-indexed")
	checkCmd.Flags().IPVar(&checkSettings.Check.Resolver, "resolver", nil, "Custom resolver to use, otherwise system default is used")

	//err := checkCmd.MarkFlagRequired("format")
	//if err != nil {
	//	rootCmd.PrintErr("Command error", err)
	//}
}

// isStdinPiped returns true if our input is from a pipe
func isStdinPiped() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}

	return isPiped(fi)
}

// isStdinPiped returns true if the output is a pipe
func isStdoutPiped() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}

	return isPiped(fi)
}

func isPiped(fi os.FileInfo) bool {
	if fi == nil {
		return false
	}

	return fi.Mode()&os.ModeNamedPipe == os.ModeNamedPipe
}
