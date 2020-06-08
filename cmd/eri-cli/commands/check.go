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
	"github.com/Dynom/ERI/types"
	"github.com/Dynom/ERI/validator"
	"github.com/Dynom/ERI/validator/validations"
	"github.com/spf13/cobra"
)

var (
	checkSettings = &CheckSettings{}
)

// checkCmd represents the check command
var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Validate email addresses",
	Long: `Check runs incrementally expensive checks on an email address
Some examples:

  - eri-cli check john.doe@example.org
  - cat list.csv | eri-cli check
  - echo "copy (select email from users) to STDOUT WITH CSV" | \
    psql <connection string> | \
    eri-cli check --resolver=8.8.8.8 | \
    tee result.json | \
    eri-cli report --only-invalid > report.json
`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return errors.New("too many arguments, expected 0 or 1")
		}

		if len(args) > 0 && isStdinPiped() {
			return errors.New("can't read both from stdin and argument")
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

		var it *iterator.CallbackIterator
		if len(args) > 0 {
			it = createTextIterator(strings.NewReader(args[0]))
		} else if isStdinPiped() {
			switch checkSettings.Format {
			case "":
				fallthrough
			case "csv":
				it = createCSVIterator(os.Stdin)
			case "text":
				// @todo this can probably go, since the liberal CSV parser handles the default text use-case as well
				it = createTextIterator(os.Stdin)
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
		for it.Next() {
			email, err := it.Value()
			if err != nil {
				cmd.PrintErr(err)
				continue
			}

			if email == "" {
				continue
			}

			var parts types.EmailParts
			if checkSettings.Check.InputIsDomain {
				parts = types.EmailParts{
					Address: email,
					Domain:  email,
				}
			} else {
				parts, err = types.NewEmailParts(email)
				if err != nil {
					cmd.PrintErr(err)
					continue
				}
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), checkSettings.Check.TTL)
			r := doCheck(ctx, v.CheckWithLookup, parts)
			cancel()

			err = jsonEncoder.Encode(r)
			if err != nil {
				cmd.PrintErr(err)
			}
		}
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
	//checkCmd.Flags().StringVar(&checkSettings.Format, "format", "csv", "Format to read. CSV works also for unquoted emails separated with a '\\n'")
	checkCmd.Flags().Uint64Var(&checkSettings.CSV.skipRows, "csv-skip-rows", 0, "Rows to skip, useful when wanting to skip the header in CSV files")
	checkCmd.Flags().Uint64Var(&checkSettings.CSV.column, "csv-column", 0, "The column to read email addresses from, 0-indexed")
	checkCmd.Flags().IPVar(&checkSettings.Check.Resolver, "resolver", nil, "Custom DNS resolver IP (e.g.: 1.1.1.1) to use, otherwise system default is used")
	checkCmd.Flags().DurationVar(&checkSettings.Check.TTL, "ttl", 30*time.Second, "Max duration per check, e.g.: '2s' or '100ms'. When exceeded, a check is considered invalid")
	checkCmd.Flags().BoolVar(&checkSettings.Check.InputIsDomain, "input-is-domain", false, "The input is a domain-name only. Checks if the domain could be valid to receive e-mail")
}