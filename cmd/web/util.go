package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"strconv"
	"time"

	gcppubsub "cloud.google.com/go/pubsub"
	"github.com/Dynom/ERI/cmd/web/erihttp"
	"github.com/Dynom/ERI/cmd/web/hitlist"
	"github.com/Dynom/ERI/cmd/web/persister"
	"github.com/Dynom/ERI/cmd/web/pubsub"
	"github.com/Dynom/ERI/cmd/web/pubsub/gcp"
	"github.com/Dynom/ERI/runtimer"
	"github.com/Dynom/ERI/types"
	"github.com/Dynom/ERI/validator"
	"github.com/Dynom/TySug/finder"
	"google.golang.org/api/option"

	"github.com/sirupsen/logrus"

	"github.com/Dynom/ERI/cmd/web/config"
)

func confHeadersToHTTPHeaders(ch config.Headers) http.Header {
	headers := http.Header{}
	for h, v := range ch {
		headers.Add(h, v)
	}

	return headers
}

func newLogger(conf config.Config) (*logrus.Logger, *io.PipeWriter, error) {
	var err error
	logger := logrus.New()

	if conf.Server.Log.Format == config.LFJSON {
		logger.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: `15:04:05.999999`,
		})
	}

	logger.SetOutput(os.Stdout)
	level, err := logrus.ParseLevel(conf.Server.Log.Level)
	if err == nil {
		logger.SetLevel(level)
	}

	return logger, logger.WriterLevel(level), err
}

func registerProfileHandler(mux *http.ServeMux, conf config.Config) {

	if !conf.Server.Profiler.Enable {
		return
	}

	var prefix = "debug"
	if conf.Server.Profiler.Prefix != "" {
		prefix = conf.Server.Profiler.Prefix
	}

	mux.HandleFunc(`/`+prefix+`/pprof/`, pprof.Index)
	mux.HandleFunc(`/`+prefix+`/pprof/cmdline`, pprof.Cmdline)
	mux.HandleFunc(`/`+prefix+`/pprof/profile`, pprof.Profile)
	mux.HandleFunc(`/`+prefix+`/pprof/symbol`, pprof.Symbol)
	mux.HandleFunc(`/`+prefix+`/pprof/trace`, pprof.Trace)
}

func setCustomResolver(dialer *net.Dialer, host string) {
	if dialer.Resolver == nil {
		dialer.Resolver = &net.Resolver{
			PreferGo: true,
		}
	}

	dialer.Resolver.Dial = func(ctx context.Context, network, address string) (conn net.Conn, e error) {
		d := net.Dialer{}
		return d.DialContext(ctx, network, net.JoinHostPort(host, `53`))
	}
}

func deferClose(toClose io.Closer, log logrus.FieldLogger) {
	if toClose == nil {
		return
	}

	err := toClose.Close()
	if err != nil {
		if log == nil {
			fmt.Printf("error failed to close handle %s", err)
			return
		}

		log.WithError(err).Error("Failed to close handle")
	}
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

func createProxiedValidator(conf config.Config, logger logrus.FieldLogger, hitList *hitlist.HitList, myFinder *finder.Finder, pubSubSvc *gcp.PubSubSvc, pgPersist persister.Persist) validator.CheckFn {
	var dialer = &net.Dialer{}
	if conf.Server.Validator.Resolver != "" {
		setCustomResolver(dialer, conf.Server.Validator.Resolver)
	}

	val := validator.NewEmailAddressValidator(dialer)

	// Pick the validator we want to use
	checkValidator := mapValidatorTypeToValidatorFn(conf.Server.Validator.SuggestValidator, val)

	if pgPersist != nil {
		logger.Info("Adding persisting validator proxy")
		checkValidator = validatorPersistProxy(pgPersist, hitList, logger, checkValidator)
	}

	if pubSubSvc != nil {
		checkValidator = validatorNotifyProxy(pubSubSvc, hitList, logger, checkValidator)
	}

	checkValidator = validatorUpdateFinderProxy(myFinder, hitList, logger, checkValidator)
	checkValidator = validatorHitListProxy(hitList, logger, checkValidator)

	return checkValidator

}

func registerHealthHandler(mux *http.ServeMux, logger logrus.FieldLogger) {
	healthHandler := NewHealthHandler(logger)

	mux.HandleFunc("/", healthHandler)
	mux.HandleFunc("/health", healthHandler)
}

func pubSubNotificationHandler(hitList *hitlist.HitList, logger logrus.FieldLogger, myFinder *finder.Finder) gcp.NotifyFn {

	logger = logger.WithField("handler", "notification")
	return func(ctx context.Context, notification pubsub.Notification) {
		parts := types.NewEmailFromParts(notification.Data.Local, notification.Data.Domain)
		if _, exists := hitList.Has(parts); exists {
			logger.WithFields(logrus.Fields{
				"notification": notification,
			}).Debug("Ignoring notification, as it's already known")
			return
		}

		vr := validator.Result{
			Validations: notification.Data.Validations,
			Steps:       notification.Data.Steps,
		}

		err := hitList.Add(parts, vr)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"error": err,
				"data":  notification.Data,
				"ctx":   ctx.Err(),
			}).Error("Unable to add to hitlist")
		}

		if vr.Validations.IsValidationsForValidDomain() && !myFinder.Exact(parts.Domain) {
			myFinder.Refresh(hitList.GetValidAndUsageSortedDomains())
		}
	}
}

func createPGPersister(conf config.Config, logger logrus.FieldLogger, hitList *hitlist.HitList) (persister.Persist, io.Closer, error) {
	if conf.Server.Backend.Driver == "" {
		logger.Info("Not setting up persistency, driver is not defined")
		return nil, nil, nil
	}

	sqlConn, err := sql.Open(conf.Server.Backend.Driver, conf.Server.Backend.URL)
	if err != nil {
		return nil, nil, err
	}

	sqlConn.SetMaxOpenConns(int(conf.Server.Backend.MaxConnections))
	sqlConn.SetMaxIdleConns(int(conf.Server.Backend.MaxIdleConnections))

	err = sqlConn.Ping()
	if err != nil {
		return nil, nil, err
	}

	p := persister.New(sqlConn, logger)
	var added uint64
	err = p.Range(context.Background(), func(d hitlist.Domain, r hitlist.Recipient, vr validator.Result) error {
		err := hitList.AddInternalParts(d, r, vr, time.Hour*60)
		if err != nil {
			logger.WithError(err).Warn("Unable hydrate hitList")
		}

		added++
		return nil
	})

	if err != nil {
		logger.WithError(err).Warn("Unable Range the database")
		return nil, nil, err
	}

	logger.WithField("added", added).Info("Hydrated hitList")
	return p, sqlConn, nil

}

func createPubSubSvc(conf config.Config, logger logrus.FieldLogger, rt *runtimer.SignalHandler, hitList *hitlist.HitList, myFinder *finder.Finder) (*gcp.PubSubSvc, error) {
	if conf.Server.GCP.PubSubTopic == "" {
		logger.Info("Not setting up pub/sub connection, no Topic defined")
		return nil, nil
	}

	psClientCtx, psClientCtxCancel := context.WithCancel(context.Background())
	psClient, err := gcppubsub.NewClient(
		psClientCtx,
		conf.Server.GCP.ProjectID,
		option.WithUserAgent("eri-"+Version),
		option.WithCredentialsFile(conf.Server.GCP.CredentialsFile),
	)

	if err != nil {
		psClientCtxCancel()
		return nil, err
	}

	pubSubSvc := gcp.NewPubSubSvc(
		logger,
		psClient,
		conf.Server.GCP.PubSubTopic,
		gcp.WithSubscriptionLabels([]string{conf.Server.InstanceID, Version, strconv.FormatInt(time.Now().Unix(), 10)}),
		gcp.WithSubscriptionConcurrencyCount(5),
	)

	// Setting up listening to notifications
	pubSubCtx, cancel := context.WithCancel(context.Background())

	rt.RegisterCallback(func(s os.Signal) {
		logger.Printf("Captured signal: %v. Starting cleanup", s)
		logger.Debug("Canceling pub/sub context")
		cancel()
	})

	rt.RegisterCallback(func(s os.Signal) {
		logger.Debug("Closing Pub/Sub service")
		deferClose(pubSubSvc, logger)
	})

	rt.RegisterCallback(func(s os.Signal) {
		logger.Debug("Canceling GCP client context")
		psClientCtxCancel()
	})

	logger.Debug("Starting listener...")
	err = pubSubSvc.Listen(pubSubCtx, pubSubNotificationHandler(hitList, logger, myFinder))
	if err != nil {
		return nil, err
	}

	return pubSubSvc, nil
}

// writeErrorJSONResponse Sets the error on a response and writes it with the corresponding Content-Type
func writeErrorJSONResponse(logger logrus.FieldLogger, w http.ResponseWriter, responseType erihttp.ERIResponse) {

	responseType.PrepareResponse()
	response, err := json.Marshal(responseType)
	if err != nil {
		logger.WithError(err).Error("Failed to marshal the response")
		response = []byte(`{"error":""}`)
	}

	w.Header().Set("Content-Type", "application/json")
	c, err := w.Write(response)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error":         err,
			"bytes_written": c,
		}).Error("Failed to write response")
		return
	}
}
