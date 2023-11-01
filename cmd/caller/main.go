package main

import (
	"fmt"
	"go.elastic.co/apm"
	"io"
	"net/http"
	"time"

	"github.com/alexthemayers/elasticfun/pkg/observability"
	"go.uber.org/zap"
)

const (
	serviceName    = "caller"
	serviceVersion = "0.0.1"
	callInterval   = 1 // seconds
)

func main() {
	tracer := observability.MustBuildNewTracer(serviceName, serviceVersion)
	defer tracer.Close()
	logger := observability.MustBuildNewLogger(tracer)
	defer logger.Sync()
	tracedClient := observability.NewTracedClient()

	// Replace with the actual URL of the second Golang service
	serviceURL := "http://middleman:8190"
	logger.Info("starting", zap.String("serviceURL", serviceURL), zap.Int("call_interval", callInterval))

	go doWork(tracer, logger, tracedClient, serviceURL)
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
