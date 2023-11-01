package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/alexthemayers/elasticfun/pkg/observability"
	"github.com/gin-gonic/gin"
	"go.elastic.co/apm/module/apmgin"
	"go.elastic.co/apm/module/apmhttp"
	"go.uber.org/zap"
	"io"
	"math/rand"
	"net/http"
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
	// FIXME: make this server switch cli-driven
	if true {
		mux := http.NewServeMux()
		mux.HandleFunc("/rng", newRngHandler(logger, tracedClient, url))

		tracedMux := apmhttp.Wrap(mux, apmhttp.WithTracer(tracer), apmhttp.WithPanicPropagation())

		// Start the custom HTTP server
		if err := http.ListenAndServe(listenPort, tracedMux); !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("Error starting stdlib HTTP server", zap.Error(err))
		}
	} else {
		engine := gin.New()
		tracingMiddlewareFunc := apmgin.Middleware(engine, apmgin.WithTracer(tracer))
		engine.Use(tracingMiddlewareFunc)
		engine.GET("/rng", newRngGinHandler(logger, tracedClient, url))
		if err := engine.Run(fmt.Sprintf("localhost%s", listenPort)); err != nil {
			logger.Fatal("Error starting gin HTTP server", zap.Error(err))
		}
	}
}

func newRngGinHandler(logger *zap.Logger, client *http.Client, url string) gin.HandlerFunc {
	return func(c *gin.Context) {
		data, err := doWork(c.Request.Context(), client, url)
		if err != nil {
			c.Error(err)
			c.JSON(http.StatusInternalServerError, nil)
			logger.Error("could not do work", zap.Error(err))
			return
		}
		// Respond to the request
		c.JSON(http.StatusOK, data)
		logger.Info("Great success")
	}
}

func newRngHandler(logger *zap.Logger, client *http.Client, url string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		testData, err := doWork(r.Context(), client, url)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			logger.Error("could not do work", zap.Error(err))
			return
		}
		data, err := json.Marshal(testData)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			logger.Error("could not prep test data for transmission", zap.Error(err))
			return
		}
		// Respond to the request
		w.WriteHeader(http.StatusOK)
		_, writeErr := w.Write(data)
		if writeErr != nil {
			logger.Error("could not write to reponse writer", zap.Error(writeErr))
		}
		logger.Info("Great success")
	}
}

func doWork(ctx context.Context, client *http.Client, url string) (TestData, error) {
	workResult := rand.Int()
	if workResult%5 == 1 {
		// fail
		return TestData{}, fmt.Errorf("failure to do work")
	}
	// succeed
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return TestData{}, fmt.Errorf("failure to make api call: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return TestData{}, fmt.Errorf("failure to make api call: %w", err)
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return TestData{}, fmt.Errorf("failure to make api call: %w", err)
	}
	var activityStruct Activity
	err = json.Unmarshal(bodyBytes, &activityStruct)
	if err != nil {
		return TestData{}, fmt.Errorf("failure to make api call: %w", err)
	}
	return TestData{Activity: activityStruct.Activity, RandomNumber: workResult}, nil
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

type TestData struct {
	Activity     string `json:"activity"`
	RandomNumber int    `json:"random_number"`
}
