# Stage 1: Build the application
FROM golang:1.22-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum to download dependencies first
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire source code
COPY . .

# Build the task_ingestion_service binary
RUN CGO_ENABLED=0 go build -o /app/task_ingestion_service ./backend/go/cmd/task_ingestion_service

# Stage 2: Create the final, minimal image
FROM alpine:latest

WORKDIR /app

# Copy the static binary from the builder stage
COPY --from=builder /app/task_ingestion_service .

# Copy the configuration file
COPY backend/go/internal/config/config.yaml .

# Expose the port the service will run on (8081 according to config)
EXPOSE 8081

# Command to run the service
CMD ["./task_ingestion_service"]