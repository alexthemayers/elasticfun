package main

import (
	"errors"
	"fmt"
	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmzap"
	"io"
	"net/http"
	"os"

	"go.elastic.co/apm/module/apmhttp"

	"go.uber.org/zap"
)

const (
	serviceName    = "middleman"
	serviceVersion = "0.0.1"
	listenPort     = ":8190"
)

func main() {
	logger := mustBuildNewLogger()
	defer logger.Sync()
	tracer := mustBuildNewTracer()
	defer tracer.Close()

	receiverURL := "http://receiver:8191"
	destinationUrl := fmt.Sprintf("%s/%s", receiverURL, "rng")
	middlemanHandler := newMiddlemanHandler(newTracedClient(), tracer, logger, destinationUrl)
	server := &http.Server{
		Addr:    listenPort,
		Handler: http.HandlerFunc(middlemanHandler),
	}

	logger.Info("Middleman service listening", zap.String("listen_port", listenPort))
	// Start the custom HTTP server
	if listenErr := server.ListenAndServe(); !errors.Is(listenErr, http.ErrServerClosed) {
		logger.Error("Error starting HTTP server", zap.Error(listenErr))
	}
}

func mustBuildNewLogger() *zap.Logger {
	//config := zap.NewProductionConfig()
	//config.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	//config.EncoderConfig.TimeKey = "@timestamp"
	//logger, loggerErr := config.Build()
	//if loggerErr != nil {
	//	fmt.Println("Error setting up Zap logger:", loggerErr)
	//	os.Exit(1)
	//}
	tracer, tracerErr := apm.NewTracerOptions(apm.TracerOptions{
		ServiceName:        serviceName,
		ServiceVersion:     serviceVersion,
		ServiceEnvironment: "local",
	})
	if tracerErr != nil {
		fmt.Printf("tracer init error encountered: %v\n", tracerErr)
		os.Exit(1)
	}
	apmcore := &apmzap.Core{Tracer: tracer}
	log, logErr := zap.NewProductionConfig().Build(zap.AddCaller())
	if logErr != nil {
		fmt.Printf("logger init error encountered: %v\n", logErr)
		os.Exit(1)
	}
	logger := zap.New(log.Core(), zap.WrapCore(apmcore.WrapCore))
	return logger
}

func newMiddlemanHandler(client *http.Client, tracer *apm.Tracer, logger *zap.Logger, externalURL string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Simulate processing by making an HTTP call to another service
		// Replace "service_url" with the actual URL of the service you want to call
		tx := tracer.StartTransaction(fmt.Sprintf("%s GET", serviceName), "request")
		defer tx.End()
		resp, err := client.Get(externalURL)
		if err != nil {
			logger.Error("Error making HTTP request", zap.Error(err))
			http.Error(w, "Error making HTTP request", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		// Check the response status code
		if resp.StatusCode != http.StatusOK {
			http.Error(w, fmt.Sprintf("status code error: secondary call returned %d", resp.StatusCode), http.StatusTeapot)
			logger.Error("HTTP call unsuccessful", zap.String("url", externalURL), zap.Int("status_code", resp.StatusCode))
			logger.Error("Request failed to process successfully", zap.String("reason", "secondary call failure"))
			return
		}

		// Respond to the incoming request
		w.WriteHeader(http.StatusOK)

		data, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			logger.Error("could not read response body from request response")
			return
		}

		_, writeErr := w.Write(data)
		if writeErr != nil {
			logger.Error("could not write response body to response writer")
			return
		}
		logger.Info("Request processed successfully")
	}
}

func newTracedClient() *http.Client {
	return &http.Client{
		Transport: apmhttp.WrapRoundTripper(http.DefaultTransport),
	}
}
func mustBuildNewTracer() *apm.Tracer {
	tracer, tracerErr := apm.NewTracer(serviceName, serviceVersion)
	if tracerErr != nil {
		fmt.Printf("Error setting up Elastic APM: %v\n", tracerErr)
		os.Exit(1)
	}
	return tracer
}