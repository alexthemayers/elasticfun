package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/alexthemayers/elasticfun/pkg/observability"
	"go.elastic.co/apm"
	"go.uber.org/zap"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

const (
	serviceName    = "caller"
	serviceVersion = "0.0.1"
	callInterval   = 1 // seconds

	// numWorkers defines number of go routines making secondary calls per program run
	numWorkers = 8
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
	serviceURL := "http://localhost:8190/delay"
	logger.Info("starting", zap.String("serviceURL", serviceURL), zap.Int("call_interval", callInterval))
	wg := &sync.WaitGroup{}
	wg.Add(numWorkers + 1) // +1 for errChan
	errChan := make(chan error)
	defer close(errChan)
	// TODO: set up condition that allows us to exit the for select loop.
	// 	 look into a go routine that listens for a SIGINT and cancels the context to
	//	 initiate a clean shutdown
	runCtx := context.TODO()
	go func() {
		ticker := time.NewTicker(callInterval * time.Second)
		for {
			select {
			case <-ticker.C:
				transactionName := fmt.Sprintf("%s.doWork(%d)", serviceName, numWorkers)
				tx := tracer.StartTransaction(transactionName, "request.orchestrator")
				logger.Info("started new transaction", zap.String("name", transactionName))
				defer tx.End()
				// Create a span that
				for i := 0; i < numWorkers; i++ {
					go func() {
						defer wg.Done()
						span := tx.StartSpan("GET /", "external.http", nil)
						defer span.End()
						// Embed the current transaction into the context and send with request
						ctx := apm.ContextWithTransaction(context.Background(), tx)
						err := doWork(ctx, logger, tracedClient, serviceURL)
						if err != nil {
							errChan <- err
						}
					}()
				}
			case <-runCtx.Done():
				wg.Done()
				// Leave the main go routine and exit cleanly
				return
			}
		}
	}()

	// drain error channel to relevant consumers
	go func() {
		defer wg.Done()
		for err := range errChan {
			logger.Error("doWork", zap.Error(err))
		}
	}()
	wg.Wait()
	logger.Info("exit successful")
	os.Exit(0)
}

func doWork(ctx context.Context, logger *zap.Logger, client *http.Client, serviceURL string) error {
	req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, serviceURL, nil)
	if reqErr != nil {
		return logHttpErr(logger, reqErr)
	}
	resp, getErr := client.Do(req)
	if getErr != nil {
		return logHttpErr(logger, getErr)
	}
	defer resp.Body.Close()
	data, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return logHttpErr(logger, readErr)
	}
	if resp.StatusCode == http.StatusOK {
		logger.Info("got response", zap.String("body", string(data)), zap.Int("status_code", resp.StatusCode))
		return nil
	} else {
		const msg = "unexpected http status code"
		logger.Error(msg, zap.String("url", serviceURL), zap.Int("status_code", resp.StatusCode))
		return errors.New(msg)
	}
}

func logHttpErr(logger *zap.Logger, err error) error {
	logger.Error("error performing http request", zap.Error(err))
	return err
}
