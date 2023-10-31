# Golang Services Instrumented with Elastic APM, Zap Logging, and ELK Stack

## Overview

This collection of Golang services is an exploration of how to instrument microservices with Elastic APM for distributed tracing, use structured logging with Zap for monitoring and troubleshooting, and report the trace and log data to the ELK (Elasticsearch, Logstash, Kibana) stack for effective visualization and analysis.

## Services

### 1. Caller Service

This is a simple Golang application that periodically makes HTTP calls to another service. This service is instrumented with Elastic APM to trace the HTTP calls and structured logging using Zap to provide structured log data. The trace and log data are sent to the ELK stack for analysis.

### 2. Middleman Service

This serves as an intermediary that receives HTTP calls from the Golang Service and makes outbound HTTP calls to other services. This service is instrumented with Elastic APM and Zap for tracing and structured logging. It provides structured log data and trace data to the ELK stack.

### 3. Receiver Service with Gin

The "Receiver Service" is a service built with Gin, designed to respond to incoming HTTP requests from the Middleman Service. This service is instrumented with Elastic APM for tracing and structured logging using Zap. It provides trace data and structured log data to the ELK stack.

## Purpose

This collection of services serves the following purposes:

- **Instrumentation:** Demonstrates how to instrument Golang microservices with Elastic APM for tracing and structured logging with Zap for effective monitoring and troubleshooting.

- **Distributed Tracing:** Shows how to trace requests as they move through the microservices, providing insights into request performance and latency.

- **Structured Logging:** Utilizes Zap for structured logging, enabling the collection of logs in a structured format, which is easier to manage and analyze.

- **ELK Stack Integration:** Demonstrates how to send trace and log data to the ELK stack (Elasticsearch, Logstash, Kibana) for visualization and analysis. The ELK stack is configured to receive and display the data effectively.

## Requirements

- Go (for building and running the services)
- Docker (for running services in containers)
- Elastic APM Server (for tracing, configure it as needed)
- ELK Stack (Elasticsearch, Logstash, Kibana) for log and trace data visualization (configure as needed)

## Usage

1. Build the services: Use the provided Dockerfiles and Makefile to build Docker images for the services.

2. Deploy the services: Start the services using Docker Compose, allowing them to interact as part of a microservices architecture.

3. Configure Elastic APM: Ensure that the Elastic APM agent in each service is configured to report to your APM server.

4. Configure ELK Stack: Make sure your ELK Stack is properly configured to receive and display the trace and log data from the services.

5. Monitor and Analyze: Access the Kibana dashboard to monitor and analyze the trace and log data from the services.

## License

This collection of services is available under the MIT License.

Feel free to customize and adapt this setup to your specific needs, and explore the possibilities of instrumenting and monitoring microservices using Elastic APM, Zap structured logging, and the ELK stack.
