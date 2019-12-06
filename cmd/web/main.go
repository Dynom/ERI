package main

import (
	"fmt"
	"net"
	"net/http"
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

	logger, err := newLogger(conf)
	if err != nil {
		panic(err)
	}

	logger.WithFields(logrus.Fields{
		"version": Version,
		"config":  conf,
	}).Info("Starting up...")

	mux := http.NewServeMux()
	checker := inspector.New(inspector.WithValidators(
		validators.ValidateMaxLength(conf.Client.InputLengthMax),
		validators.ValidateSyntax(&net.Dialer{}),
	))

	h, err := highwayhash.New128([]byte(conf.Server.Hash.Key))
	if err != nil {
		panic(err)
	}

	cache := hitlist.NewHitList(h, time.Hour*60)
	myFinder, err := finder.New(
		cache.GetValidAndUsageSortedDomains(),
		finder.WithLengthTolerance(0.2),
		finder.WithAlgorithm(finder.NewJaroWinklerDefaults()),
	)

	if err != nil {
		panic(err)
	}

	checkSvc := services.NewCheckService(&cache, myFinder, checker, logger)
	learnSvc := services.NewLearnService(&cache, myFinder, logger)

	mux.HandleFunc("/", NewHealthHandler(logger))
	mux.HandleFunc("/health", NewHealthHandler(logger))

	mux.HandleFunc("/check", NewCheckHandler(logger, checkSvc))
	mux.HandleFunc("/learn", NewLearnHandler(logger, learnSvc))

	// Debug
	mux.HandleFunc("/dumphl", func(w http.ResponseWriter, r *http.Request) {

		var domains = make([]string, 0, len(cache.Set))
		for d := range cache.Set {
			domains = append(domains, d)
		}

		sort.Strings(domains)
		for _, domain := range domains {
			_, _ = fmt.Fprintf(w, "%016b | %s \n", cache.Set[domain].Validations, domain)

			recipients, err := cache.GetRCPTsForDomain(domain)
			if err != nil {
				_, _ = fmt.Fprintf(w, "err: %s\n", err)
				continue
			}

			if len(recipients) > 0 {
				sort.Slice(recipients, func(i, j int) bool {
					return recipients[i] < recipients[j]
				})
				_, _ = fmt.Fprint(w, "\tValidations      | cache ttl                 | recipient \n")

				for _, rcpt := range recipients {
					hit, err := cache.GetHit(domain, rcpt)
					if err != nil {
						_, _ = fmt.Fprintf(w, "err: %s\n", err)
						continue
					}
					_, _ = fmt.Fprintf(w, "\t%016b | %25s | %s \n", hit.Validations, time.Now().Add(hit.TTL()).Format(time.RFC3339), rcpt)
				}
			}
		}

		_, _ = fmt.Fprintf(w, "%+v\n", cache.GetValidAndUsageSortedDomains())
	})

	lw := logger.WriterLevel(logger.Level)
	defer func() {
		_ = lw.Close()
	}()

	if conf.Server.Profiler.Enable {
		configureProfiler(mux, conf)
	}

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
