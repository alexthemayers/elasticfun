# Use the official Golang image as the base image
FROM golang:1.21

# Set the working directory inside the container
WORKDIR /app

# Copy the source code into the container
COPY . .

# Build the Middleman service binary
RUN go build -o middleman main.go

# Expose the port that the service will run on
EXPOSE 8190

# Run the Middleman service
CMD ["./middleman"]

