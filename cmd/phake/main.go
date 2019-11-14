package main

import (
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strings"
)

func main() {
	var (
		numberOfAddresses int64
		domain            = "example.org"
		eriHost           = "http://localhost:1338"
	)

	flag.Int64Var(&numberOfAddresses, "num-addr", 10, "Number of e-mail addresses to generate")
	flag.Parse()

	if numberOfAddresses <= 0 {
		flag.PrintDefaults()
		os.Exit(2)
	}

	client := http.DefaultClient
	_, _ = fmt.Fprintf(os.Stderr, "Sending %d address to /learn on %s\n", numberOfAddresses, eriHost)

	const learnReq = `{"emails": [%s]}`
	const learnReqInner = `{"value": "%s", "valid": %t}`
	const batchMaxSize = 5000
	var batchSize int64
	if numberOfAddresses > batchMaxSize {
		batchSize = batchMaxSize
	} else {
		batchSize = numberOfAddresses
	}

	var result bytes.Buffer
	for batchIndex := int64(0); batchIndex < numberOfAddresses; batchIndex += batchSize {

		var toLearn = make([]string, batchSize)
		for i := int64(0); i < batchSize; i++ {

			addr := newEmailAddress(16, domain)

			toLearn[i] = addr
			result.WriteString(wrapInJSON(eriHost, addr))
		}

		var body string
		for i, addr := range toLearn {
			body += fmt.Sprintf(learnReqInner+",", addr, i%2 == 0)
		}

		req, err := http.NewRequest(http.MethodPost, eriHost+"/learn", strings.NewReader(fmt.Sprintf(learnReq, body[:len(body)-1])))
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "boom, headshot %s", err)
			os.Exit(1)
		}

		_, _ = fmt.Fprintf(os.Stderr, "Sending batch number %d/%d\n", batchIndex, numberOfAddresses)
		res, err := client.Do(req)
		if err != nil || res.StatusCode < 200 || res.StatusCode > 299 {
			_, _ = fmt.Fprintf(os.Stderr, "boom, headshot %s\n%v", err, res)
			os.Exit(1)
		}
	}
	fmt.Println(result.String())

}

const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func newEmailAddress(length uint, domain string) string {
	var b = make([]byte, length)
	for i := uint(0); i < length; i++ {
		b[i] = letters[rand.Intn(len(letters))]
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
