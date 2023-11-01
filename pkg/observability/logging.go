package observability

import (
	"fmt"
	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmzap"
	"go.uber.org/zap"
	"os"
)

func MustBuildNewLogger(tracer *apm.Tracer) *zap.Logger {
	apmcore := &apmzap.Core{Tracer: tracer}
	log, logErr := zap.NewProductionConfig().Build(zap.AddCaller(), zap.AddStacktrace(zap.DebugLevel))
	if logErr != nil {
		fmt.Printf("logger init error encountered: %v\n", logErr)
		os.Exit(1)
	}
	logger := zap.New(log.Core(), zap.WrapCore(apmcore.WrapCore))
	return logger
}
