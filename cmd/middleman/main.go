package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"go.elastic.co/apm/module/apmhttp"
	"io"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/alexthemayers/elasticfun/pkg/observability"
	"go.uber.org/zap"
)

const (
	serviceName    = "middleman"
	serviceVersion = "0.0.1"
	listenPort     = ":8190"
	// maxWorkers defines number of go routines making secondary calls per program run
	maxWorkers = 8
	maxLatency = 1000 // milliseconds
)

func main() {
	tracer := observability.MustBuildNewTracer(serviceName, serviceVersion)
	defer tracer.Close()
	logger := observability.MustBuildNewLogger(tracer)
	defer logger.Sync()
	tracedClient := observability.NewTracedClient()

	receiverURL := "http://localhost:8191"
	fullUrl := fmt.Sprintf("%s/%s", receiverURL, "rng")
	mux := http.NewServeMux()
	mux.HandleFunc("/", newMiddlemanHandler(tracedClient, logger, fullUrl))
	mux.HandleFunc("/delay", newDelayHandler(tracedClient, logger, fullUrl))
	server := &http.Server{
		Addr:    listenPort,
		Handler: apmhttp.Wrap(mux, apmhttp.WithTracer(tracer), apmhttp.WithPanicPropagation()),
	}

	logger.Info("Middleman service listening", zap.String("listen_port", listenPort))
	// Start the custom HTTP server
	if listenErr := server.ListenAndServe(); !errors.Is(listenErr, http.ErrServerClosed) {
		logger.Error("Error starting HTTP server", zap.Error(listenErr))
	}
}

func newMiddlemanHandler(client *http.Client, logger *zap.Logger, externalURL string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Simulate processing by making an HTTP call to another service
		// Replace "service_url" with the actual URL of the service you want to call
		//tx := tracer.StartTransaction(fmt.Sprintf("%s GET", serviceName), "request")
		//defer tx.End()
		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, externalURL, nil)
		if err != nil {
			logger.Error("Error making HTTP request", zap.Error(err))
			http.Error(w, "Error making HTTP request", http.StatusInternalServerError)
			return
		}
		resp, err := client.Do(req)
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
func newDelayHandler(client *http.Client, logger *zap.Logger, externalURL string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var receiverDelays []int
		wg := &sync.WaitGroup{}
		wg.Add(maxWorkers)
		for i := 0; i < maxWorkers; i++ {
			go func() {
				wg.Done()
				receiverDelay, err := doDelayRequest(r, w, client, externalURL)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				receiverDelays = append(receiverDelays, receiverDelay)
			}()
		}
		wg.Wait()
		middlemanDelay := rand.Int63n(maxLatency)
		time.Sleep(time.Duration(middlemanDelay * int64(time.Millisecond)))
		w.WriteHeader(http.StatusOK)
		// todo: average out the receiver delays
		outDelay := receiverDelays[0] + int(middlemanDelay)
		data, err := json.Marshal(Delay{Delay: outDelay})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, err = w.Write(data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

type Delay struct {
	Delay int `json:"delay"`
}

func doDelayRequest(r *http.Request, w http.ResponseWriter, client *http.Client, externalURL string) (int, error) {
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, externalURL, nil)
	if err != nil {
		return 0, err
	}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Error making HTTP request", http.StatusInternalServerError)
		return 0, err
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("status code error: secondary call returned %d", resp.StatusCode), http.StatusTeapot)
		return 0, errors.New(fmt.Sprintf("unexpected http status code: %d", resp.StatusCode))
	}

	data, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return 0, readErr
	}
	var delayStruct Delay
	jsonErr := json.Unmarshal(data, &delayStruct)
	if jsonErr != nil {
		return 0, jsonErr
	}

	return delayStruct.Delay, nil
}
