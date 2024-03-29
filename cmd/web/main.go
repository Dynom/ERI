package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/Dynom/ERI/cmd/web/preferrer"
	"github.com/Dynom/ERI/cmd/web/pubsub/gcp"
	"github.com/Dynom/ERI/runtimer"
	"github.com/rs/cors"

	"github.com/Pimmr/rig"

	"github.com/Dynom/ERI/cmd/web/hitlist"
	"github.com/minio/highwayhash"

	"github.com/juju/ratelimit"

	"github.com/Dynom/ERI/cmd/web/services"

	"github.com/Dynom/ERI/cmd/web/config"

	"github.com/Dynom/ERI/cmd/web/erihttp"

	"github.com/Dynom/ERI/cmd/web/erihttp/handlers"

	"github.com/Dynom/TySug/finder"
	"github.com/sirupsen/logrus"

	gqlHandler "github.com/graphql-go/handler"

	_ "github.com/lib/pq"
)

// Version contains the app version, the value is changed during compile time to the appropriate Git tag
var Version = "dev"

func main() {
	var conf config.Config
	var err error
	exitCode := 0

	defer func() {
		v := recover()
		if v != nil {
			fmt.Println(v)
		}

		os.Exit(exitCode)
	}()

	conf, err = config.NewConfig("config.toml")
	if err != nil {
		exitCode = ErrExConfig
		panic(err)
	}

	err = rig.ParseStruct(&conf)
	if err != nil {
		exitCode = ErrExConfig
		panic(err)
	}

	var logger logrus.FieldLogger
	var logWriter *io.PipeWriter
	logger, logWriter, err = newLogger(conf)
	if err != nil {
		exitCode = ErrExUnavailable
		panic(err)
	}

	defer deferClose(logWriter, nil)

	logger = logger.WithField("version", Version)
	if conf.Server.InstanceID != "" {
		logger = logger.WithField("instance_id", conf.Server.InstanceID)
	}

	logger.WithFields(logrus.Fields{
		"config": conf.String(),
	}).Info("Starting up...")

	h, err := highwayhash.New128([]byte(conf.Hash.Key))
	if err != nil {
		logger.WithError(err).Error("Unable to create our hash.Hash")
		exitCode = ErrExConfig
		runtime.Goexit()
	}

	hitList := hitlist.New(
		h,
		time.Hour*60, // @todo figure out what todo with TTLs
	)

	persister, err := createPersister(conf, logger, hitList)
	if err != nil {
		logger.WithError(err).Error("Unable to setup PG persister")
		exitCode = ErrExUnavailable
		runtime.Goexit()
	}

	defer deferClose(persister, logger)

	myFinder, err := finder.New(
		hitList.GetValidAndUsageSortedDomains(),
		finder.WithLengthTolerance(conf.Finder.LengthTolerance),
		finder.WithAlgorithm(finder.NewJaroWinklerDefaults()),
		finder.WithPrefixBuckets(conf.Finder.UseBuckets),
	)
	if err != nil {
		logger.WithError(err).Error("Unable to create Finder")
		exitCode = ErrExUnavailable
		runtime.Goexit()
	}

	rtPubSub := runtimer.New(os.Interrupt, os.Kill)
	rtWeb := runtimer.New(os.Interrupt, os.Kill)

	var pubSubSvc *gcp.PubSubSvc
	pubSubSvc, err = createPubSubSvc(conf, logger, rtPubSub, hitList, myFinder)

	if err != nil {
		logger.WithError(err).Error("Unable to create the pub/sub client")
		exitCode = ErrExUnavailable
		runtime.Goexit()
	}

	prefer := preferrer.New(preferrer.Mapping(conf.Services.Suggest.Prefer))

	validatorFn := createProxiedValidator(conf, logger, hitList, myFinder, pubSubSvc, persister)
	suggestSvc := services.NewSuggestService(myFinder, validatorFn, prefer, logger)
	autocompleteSvc := services.NewAutocompleteService(myFinder, hitList, conf.Services.Autocomplete.RecipientThreshold, logger)

	mux := http.NewServeMux()
	registerProfileHandler(mux, conf)
	registerHealthHandler(mux, logger)

	mux.HandleFunc("/suggest", NewSuggestHandler(logger, suggestSvc, conf.Server.MaxRequestSize, nil))
	mux.HandleFunc("/autocomplete", NewAutoCompleteHandler(logger, autocompleteSvc, conf.Services.Autocomplete.MaxSuggestions, conf.Server.MaxRequestSize, nil))

	schema, err := NewGraphQLSchema(conf, suggestSvc, autocompleteSvc)
	if err != nil {
		logger.WithError(err).Error("Unable to build schema")
		exitCode = ErrExUnavailable
		runtime.Goexit()
	}

	mux.Handle("/graph", gqlHandler.New(&gqlHandler.Config{
		Schema:     &schema,
		Pretty:     conf.GraphQL.PrettyOutput,
		GraphiQL:   conf.GraphQL.GraphiQL,
		Playground: conf.GraphQL.Playground,
	}))

	// @todo status endpoint (or tick logger)

	var bucket handlers.TakeMaxDuration
	if conf.RateLimiter.Rate > 0 && conf.RateLimiter.Capacity > 0 {
		bucket = ratelimit.NewBucketWithRate(float64(conf.RateLimiter.Rate), conf.RateLimiter.Capacity)
	}

	ct := cors.New(cors.Options{
		AllowedOrigins: conf.Server.CORS.AllowedOrigins,
		AllowedHeaders: conf.Server.CORS.AllowedHeaders,
	})

	s := erihttp.NewServer(mux, conf, logger, logWriter, rtWeb,
		handlers.WithPathStrip(logger, conf.Server.PathStrip),
		handlers.WithRateLimiter(logger, bucket, conf.RateLimiter.ParkedTTL.AsDuration()),
		handlers.WithRequestLogger(logger),
		handlers.WithGzipHandler(),
		handlers.WithHeaders(confHeadersToHTTPHeaders(conf.Server.Headers)),
		ct.Handler,
	)

	logger.WithFields(logrus.Fields{
		"listen_on": conf.Server.ListenOn,
	}).Info("Done, serving requests")

	err = s.ServeERI()
	logger.Errorf("HTTP server stopped %s", err)

	rtPubSub.Wait()
	rtWeb.Wait()
}
