package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
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

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/handler"
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

	suggestionType := graphql.NewObject(graphql.ObjectConfig{
		Name: "suggestion",
		Fields: graphql.Fields{
			"alternatives": &graphql.Field{
				Description: "The list of alternatives. If no better match is found, the input is returned. 1 or more.",
				Type:        graphql.NewList(graphql.String),
			},

			"malformedSyntax": &graphql.Field{
				Description: "Boolean value that when true, means the address can't be valid. Conversely when false, doesn't mean it is.",
				Type:        graphql.Boolean,
			},
		},
		Description: "",
	})

	fields := graphql.Fields{
		"suggestion": &graphql.Field{
			Type: suggestionType,
			Args: graphql.FieldConfigArgument{
				"email": &graphql.ArgumentConfig{
					Type:        graphql.String,
					Description: "The e-mail address you'd like to get suggestions for",
				},
			},
			Resolve: func(p graphql.ResolveParams) (i interface{}, err error) {
				if value, ok := p.Args["email"]; ok {
					var err error
					email := value.(string)
					result, sugErr := suggestSvc.Suggest(p.Context, email)
					if sugErr != nil && sugErr != validator.ErrEmailAddressSyntax {
						err = sugErr
					}

					return erihttp.SuggestResponse{
						Alternatives:    result.Alternatives,
						MalformedSyntax: sugErr == validator.ErrEmailAddressSyntax,
					}, err
				}

				return nil, errors.New("missing required parameters")
			},
			Description: "Get suggestions",
		},
	}

	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name:   "RootQuery",
			Fields: fields,
		}),
	})

	if err != nil {
		logger.WithError(err).Error("Unable to build schema")
		os.Exit(1)
	}

	mux.Handle("/graph", handler.New(&handler.Config{
		Schema:     &schema,
		Pretty:     conf.Server.GraphQL.PrettyOutput,
		GraphiQL:   conf.Server.GraphQL.GraphiQL,
		Playground: conf.Server.GraphQL.Playground,
	}))

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
