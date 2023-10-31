# Use the official Golang image as the base image
FROM golang:1.21

# Set the working directory inside the container
WORKDIR /app

# Copy the source code into the container
COPY . .

# Build the Golang service binary
RUN go build -o caller main.go

# Run the Golang service
CMD ["./caller"]

