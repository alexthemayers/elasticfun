package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"

	"github.com/alexthemayers/elasticfun/pkg/observability"
	"go.uber.org/zap"
)

const (
	serviceName    = "receiver"
	serviceVersion = "0.0.1"
	listenPort     = ":8191"
)

func main() {
	tracer := observability.MustBuildNewTracer(serviceName, serviceVersion)
	defer tracer.Close()
	logger := observability.MustBuildNewLogger(tracer)
	defer logger.Sync()
	tracedClient := observability.NewTracedClient()

	url := "https://www.boredapi.com/api/activity"
	// Create a custom HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/rng", newRngHandler(logger, tracedClient, url))

	// Start the custom HTTP server
	if err := http.ListenAndServe(listenPort, mux); !errors.Is(err, http.ErrServerClosed) {
		logger.Error("Error starting HTTP server", zap.Error(err))
	}
}

func newRngHandler(logger *zap.Logger, client *http.Client, url string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Create a transaction for this request
		//tx := r.Tracer().StartTransaction(serviceName, "database_retrieval")
		//defer tx.End()
		//span := tx.StartSpanOptions("hard work", "rng", apm.SpanOptions{Start: time.Now()})
		//defer span.End()
		workResult := rand.Int()
		if workResult%5 == 1 {
			// fail
			http.Error(w, "failure to do work", http.StatusInternalServerError)
			logger.Error("failed to do work")
			return
		}
		// succeed
		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, url, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			logger.Error(err.Error())
			return
		}

		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			logger.Error(err.Error())
			return
		}
		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			logger.Error(err.Error())
			return
		}
		var activityStruct Activity
		err = json.Unmarshal(data, &activityStruct)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			logger.Error(err.Error())
			return
		}

		// Respond to the request
		w.WriteHeader(http.StatusOK)
		_, writeErr := w.Write([]byte(fmt.Sprintf(`{"random_number":%d, "activity": "%s"}`, workResult, activityStruct.Activity)))
		if writeErr != nil {
			logger.Error("could not write to reponse writer", zap.Error(writeErr))
		}
		logger.Info("Great success")

	}
}

type Activity struct {
	Activity      string  `json:"activity"`
	Type          string  `json:"type"`
	Participants  int     `json:"participants"`
	Price         float64 `json:"price"`
	Link          string  `json:"link"`
	Key           string  `json:"key"`
	Accessibility float64 `json:"accessibility"`
}
