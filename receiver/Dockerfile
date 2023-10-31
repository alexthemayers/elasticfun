# Use the official Golang image as the base image
FROM golang:1.21

# Set the working directory inside the container
WORKDIR /app

# Copy the source code into the container
COPY . .

# Build the Receiver service binary
RUN go build -o receiver stdhttp.go

# Expose the port that the service will run on
EXPOSE 8191

# Run the Receiver service
CMD ["./receiver"]

