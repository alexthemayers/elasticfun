package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"go.elastic.co/apm/module/apmhttp"
	"go.elastic.co/apm/module/apmzap"
	"io"
	"math/rand"
	"net/http"
	"os"
	"time"

	"go.elastic.co/apm"
	"go.uber.org/zap"
)

const (
	serviceName    = "receiver"
	serviceVersion = "0.0.1"
	listenPort     = ":8191"
)

func main() {
	logger := mustBuildLogger()
	defer logger.Sync()

	// Set up Elastic APM
	tracer, err := apm.NewTracer("", "") // Use defaults or configure as needed
	if err != nil {
		fmt.Println("Error setting up Elastic APM:", err)
		return
	}
	defer tracer.Close()

	// Define an HTTP request handling function
	rngHandler := newRNGHandler(tracer, logger)

	// Create a custom HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/rng", rngHandler)

	// Start the custom HTTP server
	if err := http.ListenAndServe(listenPort, mux); !errors.Is(err, http.ErrServerClosed) {
		logger.Error("Error starting HTTP server", zap.Error(err))
	}
}

func newRNGHandler(tracer *apm.Tracer, logger *zap.Logger) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Create a transaction for this request
		tx := tracer.StartTransaction(serviceName, "database_retrieval")
		defer tx.End()
		span := tx.StartSpanOptions("hard work", "rng", apm.SpanOptions{Start: time.Now()})
		defer span.End()
		workResult := rand.Int()
		if workResult%5 == 1 {
			// fail
			http.Error(w, "failure to do work", http.StatusInternalServerError)
			logger.Error("failed to do work")
			return
		}
		// succeed
		activityUrl := "https://www.boredapi.com/api/activity"
		resp, err := newTracedClient().Get(activityUrl)
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

func mustBuildLogger() *zap.Logger {
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

type Activity struct {
	Activity      string  `json:"activity"`
	Type          string  `json:"type"`
	Participants  int     `json:"participants"`
	Price         float64 `json:"price"`
	Link          string  `json:"link"`
	Key           string  `json:"key"`
	Accessibility float64 `json:"accessibility"`
}

func newTracedClient() *http.Client {
	return &http.Client{
		Transport: apmhttp.WrapRoundTripper(http.DefaultTransport),
	}
}
