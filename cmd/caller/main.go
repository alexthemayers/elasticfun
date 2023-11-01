package main

import (
	"context"
	"fmt"
	"github.com/alexthemayers/elasticfun/pkg/observability"
	"go.elastic.co/apm"
	"go.uber.org/zap"
	"io"
	"net/http"
	"time"
)

const (
	serviceName    = "caller"
	serviceVersion = "0.0.1"
	callInterval   = 1 // seconds
)

func main() {
	// New tracer, configured with service details for service identification
	tracer := observability.MustBuildNewTracer(serviceName, serviceVersion)
	defer tracer.Close()
	// Structured logger
	// TODO get these logs into Kibana - explore various methods for this
	logger := observability.MustBuildNewLogger(tracer)
	defer logger.Sync()
	// TOOD: This may not be necessary, given that we create a traced transaction, embed it in the context, and send
	// that with the request payload
	tracedClient := observability.NewTracedClient()

	// Replace with the actual URL of the second Golang service
	serviceURL := "http://localhost:8190"
	logger.Info("starting", zap.String("serviceURL", serviceURL), zap.Int("call_interval", callInterval))

	go func() {
		ticker := time.NewTicker(callInterval * time.Second)
		for {
			select {
			case <-ticker.C:
				// Set up tracing
				// TODO: when to use transaction, when to use span?

				// Create a transaction to represent work across multiple service boundaries
				tx := tracer.StartTransaction(fmt.Sprintf("%s.doWork", serviceName), "request")
				defer tx.End()
				// Create a span that
				span := tx.StartSpan("GET /", "external.http", nil)
				defer span.End()
				// Embed the current transaction into the context and send with request
				ctx := apm.ContextWithTransaction(context.Background(), tx)
				doWork(ctx, logger, tracedClient, serviceURL)
			}
		}
	}()
	select {}
}

func doWork(ctx context.Context, logger *zap.Logger, client *http.Client, serviceURL string) {
	// Make an HTTP GET request to the service
	req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, serviceURL, nil)
	if reqErr != nil {
		logger.Error("Error making HTTP request", zap.Error(reqErr))
		return
	}
	resp, getErr := client.Do(req)
	if getErr != nil {
		logger.Error("Error making HTTP request", zap.Error(getErr))
		return
	}
	// TODO: is this necessary if a body isn't opened?
	defer resp.Body.Close()

	// Try to read the body
	data, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		logger.Error("Error making HTTP request", zap.Error(readErr))
		return
	}

	// Check the response status code
	if resp.StatusCode == http.StatusOK {
		logger.Info("got response", zap.String("body", string(data)), zap.Int("status_code", resp.StatusCode))
	} else {
		logger.Error("HTTP call was unsuccessful", zap.String("url", serviceURL), zap.Int("status_code", resp.StatusCode))
	}
}
