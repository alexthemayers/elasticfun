package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmgin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	listenPort     = ":8192"
	serviceName    = "receiver"
	serviceVersion = "0.0.1"
)

func main() {
	// Initialize Zap logger with JSON encoding
	config := zap.NewProductionConfig()
	config.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	config.EncoderConfig.TimeKey = "@timestamp"
	logger, err := config.Build()
	if err != nil {
		fmt.Println("Error setting up Zap logger:", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Set up Elastic APM
	tracer, err := apm.NewTracer(serviceName, serviceVersion) // Use defaults or configure as needed
	if err != nil {
		logger.Error("Error setting up Elastic APM:", zap.Error(err))
		return
	}
	defer tracer.Close()

	// Create a Gin router
	router := gin.Default()

	// Set up Elastic APM middleware for Gin
	router.Use(apmgin.Middleware(router, apmgin.WithTracer(tracer)))

	// Define a route for handling requests
	router.GET("/process", func(c *gin.Context) {
		// Create a transaction for this request
		tx := apmgin.Transaction(c)
		defer tx.End()

		// Simulate processing the request
		// You can add your own business logic here

		// Respond to the request
		c.JSON(http.StatusOK, gin.H{"message": "Request processed successfully"})
	})

	// Start the Gin server
	fmt.Printf("Third service listening on %s...\n", serverAddr)
	if err := router.Run(listenPort); err != nil {
		logger.Error("Error starting Gin server", zap.Error(err))
	}
}
