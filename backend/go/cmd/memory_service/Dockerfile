# Stage 1: Build the application
FROM golang:1.22-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum to download dependencies first
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire source code
COPY . .

# Build the memory_service binary
# CGO_ENABLED=0 is important for creating a static binary that runs on alpine
# -o builds the output into a specific file path
RUN CGO_ENABLED=0 go build -o /app/memory_service ./backend/go/cmd/memory_service

# Stage 2: Create the final, minimal image
FROM alpine:latest

WORKDIR /app

# Copy the static binary from the builder stage
COPY --from=builder /app/memory_service .

# Copy the configuration file
# The path inside the container will be /app/config.yaml
COPY backend/go/internal/config/config.yaml .

# Expose the port the service will run on (update if necessary)
# EXPOSE 8080

# Command to run the service
# We point to the config file copied into the container
CMD ["./memory_service"]