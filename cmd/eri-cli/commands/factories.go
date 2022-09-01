package commands

import (
	"bufio"
	"encoding/csv"
	"io"

	"github.com/Dynom/ERI/cmd/eri-cli/iterator"
)

func createTextIterator(r io.Reader) *iterator.CallbackIterator {
	scanner := bufio.NewScanner(r)

	return iterator.NewCallbackIterator(
		scanner.Scan,
		func() (string, error) {
			return scanner.Text(), nil
		},
		func() error {
			return nil
		},
	)
}

func createCSVIterator(r io.Reader) *iterator.CallbackIterator {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = int(checkSettings.CSV.column)
	reader.ReuseRecord = true

	var lastError error
	var eof bool

	if checkSettings.CSV.skipRows > 0 {
		toSkip := checkSettings.CSV.skipRows
		for ; toSkip > 0; toSkip-- {
			_, err := reader.Read()
			if err == io.EOF {
				break
			}

			if err != nil {
				lastError = err
			}
		}
	}

	return iterator.NewCallbackIterator(
		func() bool {
			return !eof
		},
		func() (string, error) {
			var value string

			record, err := reader.Read()
			if eof || err == io.EOF {
				eof = true

				if uint64(len(record)) > checkSettings.CSV.column {
					value = record[checkSettings.CSV.column]
				}

				return value, nil
			}

			if err != nil {
				return "", err
			}

			if uint64(len(record)) > checkSettings.CSV.column {
				value = record[checkSettings.CSV.column]
			}

			return value, nil
		}, func() error {
			return lastError
		},
	)
}
