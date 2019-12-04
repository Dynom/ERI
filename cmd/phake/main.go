package main

import (
	"bytes"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"unicode/utf8"
)

const batchMaxSize = 5000
const alnum = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const letters = "abcdefghijklmnopqrstuvwxyz"

type actionType int8

type LearnValue struct {
	Value string `json:"value"`
	Valid bool   `json:"valid"`
}

const (
	Generate   actionType = iota
	ReadDomain actionType = iota
	ReadEmail  actionType = iota
)

func main() {
	var (
		numberOfAddresses int64
		domain            = "example.org"
		eriHost           = "http://localhost:1338"
		generateDomain    bool
		assumeValid       bool
		domainsFromFile   string
		emailsFromFile    string
		action            = Generate
	)

	flag.Int64Var(&numberOfAddresses, "num-addr", 10, "Number of e-mail addresses to generate")
	flag.BoolVar(&generateDomain, "gen-domain", false, "Pass the flag to generate the domain name")

	flag.StringVar(&domainsFromFile, "read-domains-from", "", "Learn from pre-existing CSV file instead. ./path/to.csv")
	flag.StringVar(&emailsFromFile, "read-emails-from", "", "Learn from pre-existing CSV file instead. ./path/to.csv")
	flag.BoolVar(&assumeValid, "read-is-valid", false, "Assume all domains being read valid (true) or invalid")
	flag.Parse()

	// Implied action
	if domainsFromFile != "" {
		action = ReadDomain
	} else if emailsFromFile != "" {
		action = ReadEmail
		panic("Not yet implemented")
	}

	if action == Generate {
		if numberOfAddresses <= 0 {
			flag.PrintDefaults()
			os.Exit(2)
		}

		if generateDomain {
			domain = ""
		}

		_, _ = fmt.Fprintf(os.Stderr, "Sending %d address to /learn on %s\n", numberOfAddresses, eriHost)

		var result bytes.Buffer
		generateAndSendBatches(&result, numberOfAddresses, domain, eriHost)

		fmt.Println(result.String())
	} else if action == ReadDomain /*|| action == ReadEmail*/ {

		if domainsFromFile == "" {
			fmt.Printf("Need a CSV file name to read from\n")
			os.Exit(1)
		}

		file, err := os.OpenFile(domainsFromFile, os.O_RDONLY, 0644)
		if err != nil {
			fmt.Printf("Unable to open the file '%s'", err)
			os.Exit(1)
		}

		reader := csv.NewReader(file)
		var domains = make([]LearnValue, 0, batchMaxSize)
		for {
			row, err := reader.Read()

			if err == nil && len(row) > 0 {
				domain := row[0]

				for pos, char := range domain {
					if isInvisible(char) || !utf8.ValidRune(char) {
						fmt.Printf("\t '%s' has character: '%s' = 0x%x on position %d\n", domain, string(char), char, pos)
					}
					continue
				}

				domains = append(domains, LearnValue{
					Value: domain,
					Valid: assumeValid,
				})

				if len(domains) >= batchMaxSize {
					err := sendBatch(http.DefaultClient, domains, eriHost, action)
					if err != nil {
						fmt.Printf("Error sending batch (not aborting): %s\n", err)
					}

					// truncate domains
					domains = domains[:0] //[]string{}
				}

				continue
			}

			if err == io.EOF {
				if len(domain) > 0 {
					err := sendBatch(http.DefaultClient, domains, eriHost, action)
					if err != nil {
						fmt.Printf("Error sending batch (not aborting): %s\n", err)
					}
				}

				fmt.Printf("EOF!\n")
			}

			break
		}
	}
}

func generateAndSendBatches(result io.StringWriter, numberOfAddresses int64, domain, eriHost string) {
	var batchSize int64
	if numberOfAddresses > batchMaxSize {
		batchSize = batchMaxSize
	} else {
		batchSize = numberOfAddresses
	}

	client := http.DefaultClient
	for batchIndex := int64(0); batchIndex < numberOfAddresses; batchIndex += batchSize {

		var toLearn = make([]LearnValue, batchSize)
		for i := int64(0); i < batchSize; i++ {

			addr := newEmailAddress(16, domain)

			toLearn[i] = LearnValue{
				Value: addr,
				Valid: i%2 == 0,
			}
			_, _ = result.WriteString(wrapInJSON(eriHost, addr))
		}

		_, _ = fmt.Fprintf(os.Stderr, "Sending batch [%d/%d]\n", batchIndex, numberOfAddresses)
		err := sendBatch(client, toLearn, eriHost, Generate)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "boom, headshot %s", err)
			os.Exit(1)
		}
	}
}

func sendBatch(client *http.Client, addresses []LearnValue, eriHost string, action actionType) error {
	var learnReq string

	switch action {
	case ReadDomain:
		learnReq = `{"domains": [%s]}`
	case ReadEmail:
		fallthrough
	case Generate:
		learnReq = `{"emails": [%s]}`
	}

	var body string
	for _, addr := range addresses {
		value, err := json.Marshal(addr)
		if err != nil {
			return err
		}

		body += string(value) + ","
	}

	fmt.Printf("Sending:\n%s\n\n", fmt.Sprintf(learnReq, body[:len(body)-1]))
	req, err := http.NewRequest(http.MethodPost, eriHost+"/learn", strings.NewReader(fmt.Sprintf(learnReq, body[:len(body)-1])))
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "boom, headshot %s", err)
		os.Exit(1)
	}

	res, err := client.Do(req)
	if err != nil || res.StatusCode < 200 || res.StatusCode > 299 {
		return fmt.Errorf("wrror or bad status %w, %+v", err, res)
	}

	return nil
}

func newEmailAddress(length uint, domain string) string {
	var b = make([]byte, length)
	for i := uint(0); i < length; i++ {
		b[i] = alnum[rand.Intn(len(alnum))]
	}

	if len(domain) == 0 {
		var d = make([]byte, 20+rand.Intn(38))

		d[0] = letters[rand.Intn(len(letters))]
		for i, j := 1, len(d); i < j; i++ {
			d[i] = alnum[rand.Intn(len(alnum))]
		}

		domain = string(d) + `.test`
	}

	return string(b) + `@` + domain
}

func wrapInJSON(eriHost, emailAddr string) string {
	var vegetaTpl = `{"method": "POST", "url": "` + eriHost + `/check", "headers": {"Content-Type": "application/json"}, "body": "%s"}`
	const eriTpl = `{"email": "%s", "with_alternatives": true}`

	return fmt.Sprintf(
		vegetaTpl+"\n",
		base64.StdEncoding.EncodeToString(
			[]byte(fmt.Sprintf(eriTpl, emailAddr)),
		),
	)
}

func isInvisible(c rune) bool {
	switch {
	case 48 <= c && c <= 57 /* 0-9 */ :
	case 65 <= c && c <= 90 /* A-Z */ :
	case 97 <= c && c <= 122 /* a-z */ :
	case c == 45 /* dash - */ :
	case c == 46 /* dot . */ :
	case c == 0xa0:
	case c == 0x3e:
	default:
		return true
	}

	return false
}
