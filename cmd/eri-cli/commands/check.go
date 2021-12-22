package commands

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"strings"
	"time"

	"github.com/Dynom/ERI/cmd/eri-cli/iterator"
	"github.com/Dynom/ERI/cmd/eri-cli/werkit"
	"github.com/Dynom/ERI/types"
	"github.com/Dynom/ERI/validator"
	"github.com/Dynom/ERI/validator/validations"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	checkSettings = &CheckSettings{}
)

const (
	inputFormatText = "text"
	inputFormatCSV  = "csv"
)

// checkCmd represents the check command
var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Validate email addresses",
	Long: `Check runs incrementally expensive checks on an email address
Some examples:

  - eri-cli check john.doe@example.org
  - eri-cli check example.org
  - cat list.csv | eri-cli check
  - echo "copy (select email from users) to STDOUT WITH CSV" | \
    psql <connection string> | \
    eri-cli check --resolver=8.8.8.8 | \
    tee result.json | \
    eri-cli report --only-invalid > report.json
`,
	Args: func(cmd *cobra.Command, args []string) error {

		var stdInFromTerminal = term.IsTerminal(int(os.Stdin.Fd()))
		if len(args) > 1 {
			return errors.New("too many arguments, expected 0 or 1")
		}

		if len(args) > 0 && !stdInFromTerminal {
			return errors.New("can't read both from stdin and argument")
		}

		if len(args) == 0 && stdInFromTerminal {
			return errors.New("missing argument")
		}

		if checkSettings.Workers < 1 {
			return errors.New("minimum number of workers is 1")
		}

		if checkSettings.Workers > 1024 {
			return errors.New("maximum number of workers is 1024")
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		var dialer = &net.Dialer{}
		if checkSettings.Check.Resolver != nil {
			setCustomResolver(dialer, checkSettings.Check.Resolver)
		}

		v := validator.NewEmailAddressValidator(dialer)

		var workers = int(checkSettings.Workers)
		var it *iterator.CallbackIterator
		if len(args) > 0 {
			it = createTextIterator(strings.NewReader(args[0]))
			workers = 1
		} else {
			switch checkSettings.Format {
			case "":
				fallthrough
			case inputFormatCSV:
				it = createCSVIterator(cmd.InOrStdin())
			case inputFormatText:
				// @todo this can probably go, since the liberal CSV parser handles the default text use-case as well
				it = createTextIterator(cmd.InOrStdin())
			default:
				cmd.PrintErrf("bad format %q", checkSettings.Format)
				return
			}
		}

		if it == nil {
			cmd.PrintErr("No suitable iterator found, this is.. unexpected.")
			return
		}

		jsonEncoder := json.NewEncoder(cmd.OutOrStdout())
		wi := werkit.WerkIt{}
		wi.StartCheckWorkers(workers, func(tasks <-chan werkit.CheckTask) {
			for task := range tasks {

				ctx, cancel := context.WithTimeout(task.Ctx, checkSettings.Check.TTL)
				r := doCheck(ctx, task.Fn, task.Parts)
				cancel()

				err := jsonEncoder.Encode(r)
				if err != nil {
					cmd.PrintErr(err)
				}
			}
		})

		for it.Next() {
			email, err := it.Value()
			if err != nil {
				cmd.PrintErr(err)
				continue
			}

			if email == "" {
				continue
			}

			parts, err := types.NewEmailParts(email)
			if err != nil {
				if err == types.ErrInvalidEmailAddress && !checkSettings.Check.InputIsEmailAddress {
					parts = types.EmailParts{
						Address: email,
						Domain:  email,
					}
				} else {
					cmd.PrintErr(err)
					continue
				}
			}

			wi.Process(werkit.CheckTask{
				Ctx:   cmd.Context(),
				Fn:    v.CheckWithLookup,
				Parts: parts,
			})
		}

		wi.Wait()
	},
}

func doCheck(ctx context.Context, fn validator.CheckFn, parts types.EmailParts) CheckResultFull {
	var result = CheckResultFull{
		Input:   parts.Address,
		Version: 2,
	}

	{
		checkResult := fn(ctx, parts)

		result.Valid = checkResult.Validations.IsValid()
		result.Passed = validations.Flag(checkResult.Validations.RemoveFlag(validations.FValid)).AsStringSlice()
		result.Checks = validations.Flag(checkResult.Steps).AsStringSlice()
	}

	return result
}

func init() {
	rootCmd.AddCommand(checkCmd)

	// Disabled for now, since foo\nbar\n parses fine in the liberal CSV parser.
	// checkCmd.Flags().StringVar(&checkSettings.Format, "format", inputFormatCSV, "Format to read. CSV works also for unquoted emails separated with a '\\n'")
	checkCmd.Flags().Uint64Var(&checkSettings.CSV.skipRows, "csv-skip-rows", 0, "Rows to skip, useful when wanting to skip the header in CSV files")
	checkCmd.Flags().Uint64Var(&checkSettings.CSV.column, "csv-column", 0, "The column to read email addresses from, 0-indexed")
	checkCmd.Flags().IPVar(&checkSettings.Check.Resolver, "resolver", nil, "Custom DNS resolver IP (e.g.: 1.1.1.1) to use, otherwise system default is used")
	checkCmd.Flags().DurationVar(&checkSettings.Check.TTL, "ttl", 30*time.Second, "Max duration per check, e.g.: '2s' or '100ms'. When exceeded, a check is considered invalid")
	checkCmd.Flags().BoolVar(&checkSettings.Check.InputIsEmailAddress, "input-is-email", false, "If the input isn't an e-mail address, don't fall back on domain only checks")
	checkCmd.Flags().Uint64Var(&checkSettings.Workers, "workers", 50, "The number of concurrent workers to use when in piped mode (1-1024)")
}
