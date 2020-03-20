package main

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/Dynom/ERI/cmd/web/hitlist"
	"github.com/minio/highwayhash"

	"github.com/juju/ratelimit"

	"github.com/Dynom/ERI/validator"

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

	hitList := hitlist.New(
		h,
		time.Hour*60, // @todo figure out what todo with TTLs
	)

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

	validationResultCache := &sync.Map{}
	validationResultPersister := &sync.Map{}

	val := validator.NewEmailAddressValidator(dialer)

	// Pick the validator we want to use
	checkValidator := mapValidatorTypeToValidatorFn(conf.Server.Validator.SuggestValidator, val)

	// Wrap it
	checkValidator = validatorPersistProxy(validationResultPersister, logger, checkValidator)
	checkValidator = validatorUpdateFinderProxy(myFinder, hitList, logger, checkValidator)
	checkValidator = validatorCacheProxy(validationResultCache, logger, checkValidator)

	// Use it
	suggestSvc := services.NewSuggestService(myFinder, checkValidator, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("/", NewHealthHandler(logger))
	mux.HandleFunc("/health", NewHealthHandler(logger))

	mux.HandleFunc("/suggest", NewSuggestHandler(logger, suggestSvc))
	mux.HandleFunc("/autocomplete", NewAutoCompleteHandler(logger, myFinder))

	lw := logger.WriterLevel(logger.Level)
	defer func() {
		_ = lw.Close()
	}()

	if conf.Server.Profiler.Enable {
		configureProfiler(mux, conf)
	}

	// @todo status endpoint (or tick logger)
	// @todo make the RL configurable

	bucket := ratelimit.NewBucketWithRate(100, 500)
	s := erihttp.BuildHTTPServer(mux, conf, lw,
		//handlers.NewRateLimitHandler(logger, bucket, time.Millisecond*100),
		handlers.WithRequestLogger(logger),
		handlers.WithGzipHandler(),
		handlers.WithHeaders(sliceToHTTPHeaders(conf.Server.Headers)),
	)

	_ = bucket

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

	panic(fmt.Sprintf("Incorrect validator %q configured.", vt))
}
