package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/alexthemayers/elasticfun/cmd/receiver/testdata"
	"github.com/alexthemayers/elasticfun/pkg/observability"
	"github.com/gin-gonic/gin"
	"go.elastic.co/apm/module/apmgin"
	"go.elastic.co/apm/module/apmhttp"
	"go.uber.org/zap"
	"math/rand"
	"net/http"
	"time"
)

const (
	serviceName    = "receiver"
	serviceVersion = "0.0.1"
	listenPort     = ":8191"
	maxLatency     = 1000 // milliseconds
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
		mux.HandleFunc("/delay", newDelayHandler(logger))

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
		data, err := testdata.CallExternalApi(c.Request.Context(), client, url)
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
		testData, err := testdata.CallExternalApi(r.Context(), client, url)
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
func newDelayHandler(logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		delay := time.Duration(rand.Int63n(maxLatency) * int64(time.Millisecond))
		time.Sleep(delay)
		_, writeErr := w.Write([]byte(fmt.Sprintf(`{"delay":%d}`, delay)))
		if writeErr != nil {
			http.Error(w, writeErr.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}
}
