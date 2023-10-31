package main

import (
	"fmt"
	"go.elastic.co/apm/module/apmzap"
	"io"
	"net/http"
	"os"
	"time"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmhttp"
	"go.uber.org/zap"
)

const (
	serviceName    = "caller"
	serviceVersion = "0.0.1"
	callInterval   = 1 // seconds
)

func main() {
	logger := mustBuildNewLogger()
	defer logger.Sync()

	tracer := mustBuildNewTracer()
	defer tracer.Close()

	// Replace with the actual URL of the second Golang service
	serviceURL := "http://middleman:8190"
	logger.Info("starting", zap.String("serviceURL", serviceURL), zap.Int("call_interval", callInterval))

	client := newTracedClient()
	go doWork(tracer, logger, client, serviceURL)
	select {}
}

func doWork(tracer *apm.Tracer, logger *zap.Logger, client *http.Client, serviceURL string) {
	ticker := time.NewTicker(callInterval * time.Second)
	for {
		select {
		case <-ticker.C:
			// Set up tracing
			// TODO: when to use transaction, when to use span?
			tx := tracer.StartTransaction(fmt.Sprintf("%s: GET", serviceName), "request")
			defer tx.End()

			// Make an HTTP GET request to the service
			resp, getErr := client.Get(serviceURL)
			if getErr != nil {
				logger.Error("Error making HTTP request", zap.Error(getErr))
				continue
			}
			// TODO: is this necessary if a body isn't opened?
			defer resp.Body.Close()

			// Try to read the body
			data, readErr := io.ReadAll(resp.Body)
			if readErr != nil {
				logger.Error("Error making HTTP request", zap.Error(readErr))
				continue
			}

			// Check the response status code
			if resp.StatusCode == http.StatusOK {
				logger.Info("got response", zap.String("body", string(data)), zap.Int("status_code", resp.StatusCode))
			} else {
				logger.Error("HTTP call was unsuccessful", zap.String("url", serviceURL), zap.Int("status_code", resp.StatusCode))
			}
		}
	}
}

func newTracedClient() *http.Client {
	return &http.Client{
		Transport: apmhttp.WrapRoundTripper(http.DefaultTransport),
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
func mustBuildNewTracer() *apm.Tracer {
	tracer, tracerErr := apm.NewTracer(serviceName, serviceVersion) // Use defaults or configure as needed
	if tracerErr != nil {
		fmt.Printf("Error setting up Elastic APM: %v\n", tracerErr)
		os.Exit(1)
	}
	return tracer
}
