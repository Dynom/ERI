package main

import (
	"fmt"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/Dynom/ERI/cmd/web/inspector/validators"

	"github.com/minio/highwayhash"

	"github.com/Dynom/ERI/cmd/web/hitlist"

	"github.com/Dynom/ERI/cmd/web/services"

	"github.com/Dynom/ERI/cmd/web/config"

	"github.com/Dynom/ERI/cmd/web/erihttp"

	"github.com/Dynom/ERI/cmd/web/erihttp/handlers"

	"github.com/Dynom/ERI/cmd/web/inspector"
	"github.com/Dynom/TySug/finder"
	"github.com/sirupsen/logrus"
)

// Version contains the app version, the value is changed during compile time to the appropriate Git tag
var Version = "dev"

func main() {
	var conf config.Config
	var err error

	conf, err = config.NewConfig("config.toml")
	if err != nil {
		panic(err)
	}

	logger := logrus.New()
	logger.Formatter = &logrus.JSONFormatter{}
	logger.Out = os.Stdout
	logger.Level, err = logrus.ParseLevel(conf.Server.Log.Level)

	if err != nil {
		panic(err)
	}

	logger.WithFields(logrus.Fields{
		"version": Version,
	}).Info("Starting up...")

	mux := http.NewServeMux()

	checker := inspector.New(inspector.WithValidators(
		validators.ValidateSyntax(),
		validators.ValidateMXAndRCPT(validators.DefaultRecipient),
	))

	h, err := highwayhash.New128([]byte(`a1C2d3oi4uctnqo3utlNcwtqlmwH!rtl`))
	if err != nil {
		panic(err)
	}

	cache := hitlist.NewHitList(h)
	myFinder, err := finder.New(
		cache.GetValidAndUsageSortedDomains(),
		finder.WithLengthTolerance(0.2),
		finder.WithAlgorithm(finder.NewJaroWinklerDefaults()),
	)

	if err != nil {
		panic(err)
	}

	checkSvc := services.NewCheckService(&cache, myFinder, checker, logger)
	learnSvc := services.NewLearnService(&cache, myFinder)

	mux.HandleFunc("/check", NewCheckHandler(logger, checkSvc))
	mux.HandleFunc("/dumphl", func(w http.ResponseWriter, r *http.Request) {

		var domains = make([]string, 0, len(cache.Set))
		for d := range cache.Set {
			domains = append(domains, d)
		}

		sort.Strings(domains)
		for _, domain := range domains {
			_, _ = fmt.Fprintf(w, "%s\n", domain)

			rcpts := make([]hitlist.RCPT, 0, len(cache.Set[domain].RCPTs))
			for rcpt := range cache.Set[domain].RCPTs {
				rcpts = append(rcpts, rcpt)
			}

			if len(rcpts) > 0 {
				sort.Slice(rcpts, func(i, j int) bool {
					return rcpts[i] < rcpts[j]
				})
				_, _ = fmt.Fprint(w, "\tValidations      | cache ttl                 | recipient \n")

				for _, rcpt := range rcpts {
					hit := cache.Set[domain].RCPTs[rcpt]
					_, _ = fmt.Fprintf(w, "\t%016b | %25s | %s \n", hit.Validations, hit.ValidUntil.Format(time.RFC3339), rcpt)
				}
			}
		}

		_, _ = fmt.Fprintf(w, "%+v\n", cache.GetValidAndUsageSortedDomains())
	})
	mux.HandleFunc("/learn", NewLearnHandler(logger, learnSvc))

	lw := logger.WriterLevel(logger.Level)
	defer func() {
		_ = lw.Close()
	}()

	s := erihttp.BuildHTTPServer(mux, conf, lw,
		handlers.WithGzipHandler(),
		handlers.WithHeaders(sliceToHTTPHeaders(conf.Server.Headers)),
	)

	logger.WithFields(logrus.Fields{
		"listen_on": conf.Server.ListenOn,
	}).Info("Done, serving requests")
	err = s.ListenAndServe()

	logger.Errorf("HTTP server stopped %s", err)
}
