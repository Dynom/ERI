package main

import (
	"database/sql"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/juju/ratelimit"

	validator "github.com/Dynom/ERI/validator"

	"github.com/minio/highwayhash"

	"github.com/Dynom/ERI/cmd/web/hitlist"

	"github.com/Dynom/ERI/cmd/web/services"

	"github.com/Dynom/ERI/cmd/web/config"

	"github.com/Dynom/ERI/cmd/web/erihttp"

	"github.com/Dynom/ERI/cmd/web/erihttp/handlers"

	"github.com/Dynom/TySug/finder"
	"github.com/sirupsen/logrus"

	_ "github.com/lib/pq"
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

	hitList := hitlist.New(
		h,
		time.Hour*60, // @todo figure out what todo with TTLs
		hitlist.WithMaxCallBackConcurrency(5),
	)

	if conf.Server.Backend.Driver != "" {
		sqlConn, err := sql.Open(conf.Server.Backend.Driver, conf.Server.Backend.URL)
		if err != nil {
			panic(err)
		}

		defer deferClose(sqlConn, logger)
		collected, err := preloadValues(sqlConn, hitList, logger)
		if err != nil {
			logger.Errorf("A backend has been configured, but the connection failed %s", err)
			os.Exit(1)
		}

		logger.WithField("amount", collected).Info("pre-loaded values from the database")

		registerPersistCallback(sqlConn, hitList, logger)
		logger.Info("registered persisting callback, newly learned values will be persisted")
	}

	myFinder, err := finder.New(
		hitList.GetValidAndUsageSortedDomains(),
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
	checkSvc := services.NewCheckService(hitList, myFinder, mapValidatorTypeToValidatorFn(conf.Server.Validator.CheckValidator, val), logger)
	learnSvc := services.NewLearnService(hitList, myFinder, mapValidatorTypeToValidatorFn(conf.Server.Validator.LearnValidator, val), logger)
	suggestSvc := services.NewSuggestService(hitList, myFinder, mapValidatorTypeToValidatorFn(conf.Server.Validator.LearnValidator, val), logger)

	mux := http.NewServeMux()
	mux.HandleFunc("/", NewHealthHandler(logger))
	mux.HandleFunc("/health", NewHealthHandler(logger))

	mux.HandleFunc("/suggest", NewSuggestHandler(logger, suggestSvc))
	mux.HandleFunc("/check", NewCheckHandler(logger, checkSvc))
	mux.HandleFunc("/learn", NewLearnHandler(logger, learnSvc))
	mux.HandleFunc("/autocomplete", NewAutoCompleteHandler(logger, myFinder))

	lw := logger.WriterLevel(logger.Level)
	defer func() {
		_ = lw.Close()
	}()

	if conf.Server.Profiler.Enable {
		configureProfiler(mux, conf)
	}

	// @todo status endpoint (or tick logger)

	bucket := ratelimit.NewBucketWithRate(100, 500)
	s := erihttp.BuildHTTPServer(mux, conf, lw,
		handlers.NewRateLimitHandler(logger, bucket, time.Millisecond*100),
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

func mapValidatorTypeToValidatorFn(vt config.ValidatorType, v validator.EmailValidator) validator.CheckFn {
	switch vt {
	case config.VTLookup:
		return v.CheckWithLookup
	case config.VTStructure:
		return v.CheckWithSyntax
	}

	panic("Incorrect validator to map, this probably means an inconsistency between main and config packages.")
}
