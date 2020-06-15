package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	cli "github.com/jawher/mow.cli"
	metrics "github.com/rcrowley/go-metrics"

	fthealth "github.com/Financial-Times/go-fthealth/v1_1"
	logger "github.com/Financial-Times/go-logger/v2"
	"github.com/Financial-Times/http-handlers-go/v2/httphandlers"
	status "github.com/Financial-Times/service-status-go/httphandlers"
)

const (
	appDescription = ""
	// TODO: how long we would like to wait for response?
	httpServerReadTimeout  = 10 * time.Second
	httpServerWriteTimeout = 15 * time.Second
	httpServerIdleTimeout  = 20 * time.Second
	httpHandlersTimeout    = 14 * time.Second
)

func main() {
	app := cli.App("cm-go-service", appDescription)

	appSystemCode := app.String(cli.StringOpt{
		Name:   "app-system-code",
		Value:  "cm-go-service",
		Desc:   "system Code of the application",
		EnvVar: "APP_SYSTEM_CODE",
	})

	appName := app.String(cli.StringOpt{
		Name:   "app-name",
		Value:  "cm-go-service",
		Desc:   "application name",
		EnvVar: "APP_NAME",
	})

	port := app.String(cli.StringOpt{
		Name:   "port",
		Value:  "8080",
		Desc:   "port to listen on",
		EnvVar: "APP_PORT",
	})

	logLevel := app.String(cli.StringOpt{
		Name:   "log-level",
		Value:  "INFO",
		Desc:   "logging level (DEBUG, INFO, WARN, ERROR)",
		EnvVar: "LOG_LEVEL",
	})

	log := logger.NewUPPLogger(*appName, *logLevel)

	app.Action = func() {
		log.Infof("starting with system code: %s, app name: %s, port: %s", *appSystemCode, *appName, *port)

		healthService := NewHealthService(*appSystemCode, *appName, appDescription)

		router := registerEndpoints(healthService, log)

		server := newHTTPServer(*port, router)
		go startHTTPServer(server, log)

		waitForSignal()
		stopHTTPServer(server, log)
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Errorf("app could not start: %v", err)
		return
	}
}

func registerEndpoints(healthService *HealthService, log *logger.UPPLogger) http.Handler {
	serveMux := http.NewServeMux()

	// register supervisory endpoint that does not require logging and metrics collection
	serveMux.HandleFunc("/__health", fthealth.Handler(healthService.Health()))
	serveMux.HandleFunc(status.GTGPath, status.NewGoodToGoHandler(healthService.GTG))
	serveMux.HandleFunc(status.BuildInfoPath, status.BuildInfoHandler)

	// add services router and register endpoints specific to this service only
	servicesRouter := mux.NewRouter()
	//TODO: add real handlers
	servicesRouter.HandleFunc("/test", TestHandler).Methods("GET")

	// wrap the handler with certain middlewares providing logging of the requests,
	// sending metrics and handler time out on certain time interval
	var wrappedServicesRouter http.Handler = servicesRouter
	wrappedServicesRouter = httphandlers.TransactionAwareRequestLoggingHandler(log, wrappedServicesRouter)
	wrappedServicesRouter = httphandlers.HTTPMetricsHandler(metrics.DefaultRegistry, wrappedServicesRouter)
	wrappedServicesRouter = http.TimeoutHandler(wrappedServicesRouter, httpHandlersTimeout, "")

	serveMux.Handle("/", wrappedServicesRouter)

	return serveMux
}

func newHTTPServer(port string, router http.Handler) *http.Server {
	return &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  httpServerReadTimeout,
		WriteTimeout: httpServerWriteTimeout,
		IdleTimeout:  httpServerIdleTimeout,
	}
}

func startHTTPServer(server *http.Server, log *logger.UPPLogger) {
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("http server failed to start: %v", err)
	}
}

func stopHTTPServer(server *http.Server, log *logger.UPPLogger) {
	log.Info("http server is shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("failed to gracefully shutdown the server: %v", err)
	}
}

func waitForSignal() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
}
