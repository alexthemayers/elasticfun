package observability

import (
	"fmt"
	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmhttp"
	"net/http"
	"os"
)

func MustBuildNewTracer(serviceName, serviceVersion string) *apm.Tracer {
	apm.DefaultTracer.Close()
	tracer, tracerErr := apm.NewTracer(serviceName, serviceVersion)
	if tracerErr != nil {
		fmt.Printf("Error setting up Elastic APM: %v\n", tracerErr)
		os.Exit(1)
	}
	return tracer
}

func NewTracedClient() *http.Client {
	return apmhttp.WrapClient(&http.Client{}, apmhttp.WithClientTrace())
}
