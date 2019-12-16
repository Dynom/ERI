package main

import (
	"net"
	"net/http"
	"time"

	validator "github.com/Dynom/ERI/validator"

	"github.com/minio/highwayhash"

	"github.com/Dynom/ERI/cmd/web/hitlist"

	"github.com/Dynom/ERI/cmd/web/services"

	"github.com/Dynom/ERI/cmd/web/config"

	"github.com/Dynom/ERI/cmd/web/erihttp"

	"github.com/Dynom/ERI/cmd/web/erihttp/handlers"

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

	h, err := highwayhash.New128([]byte(conf.Server.Hash.Key))
	if err != nil {
		panic(err)
	}

	cache := hitlist.New(h, time.Hour*60) // @todo figure out what todo with TTLs
	myFinder, err := finder.New(
		cache.GetValidAndUsageSortedDomains(),
		finder.WithLengthTolerance(0.2),
		finder.WithAlgorithm(finder.NewJaroWinklerDefaults()),
		finder.WithPrefixBuckets(conf.Server.Finder.UseBuckets),
	)

	if err != nil {
		panic(err)
	}

	var dialer = &net.Dialer{}
	if conf.Server.Validator.Resolver != "" {
		setCustomResolver(dialer, conf.Server.Validator.Resolver)
	}

	val := validator.NewEmailAddressValidator(dialer)
	checkSvc := services.NewCheckService(cache, myFinder, val.CheckWithSyntax, logger)
	learnSvc := services.NewLearnService(cache, myFinder, val.CheckWithSyntax, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("/", NewHealthHandler(logger))
	mux.HandleFunc("/health", NewHealthHandler(logger))

	mux.HandleFunc("/check", NewCheckHandler(logger, checkSvc))
	mux.HandleFunc("/learn", NewLearnHandler(logger, learnSvc))
	mux.HandleFunc("/autocomplete", NewAutoCompleteHandler(logger, myFinder))

	// Debug
	mux.HandleFunc("/dumphl", NewDebugHandler(cache))

	lw := logger.WriterLevel(logger.Level)
	defer func() {
		_ = lw.Close()
	}()

	if conf.Server.Profiler.Enable {
		configureProfiler(mux, conf)
	}

	s := erihttp.BuildHTTPServer(mux, conf, lw,
		handlers.WithRequestLogger(logger),
		handlers.WithGzipHandler(),
		handlers.WithHeaders(sliceToHTTPHeaders(conf.Server.Headers)),
	)

	logger.WithFields(logrus.Fields{
		"listen_on": conf.Server.ListenOn,
	}).Info("Done, serving requests")
	err = s.ListenAndServe()

	logger.Errorf("HTTP server stopped %s", err)
}
