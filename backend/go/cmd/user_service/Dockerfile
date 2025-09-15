# Stage 1: Build the application
FROM golang:1.22-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum to download dependencies first
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire source code
COPY . .

# Build the user_service binary
RUN CGO_ENABLED=0 go build -o /app/user_service ./backend/go/cmd/user_service

# Stage 2: Create the final, minimal image
FROM alpine:latest

WORKDIR /app

# Copy the static binary from the builder stage
COPY --from=builder /app/user_service .

# Copy the configuration file
COPY backend/go/internal/config/config.yaml .

# Expose the port the service will run on (update if necessary)
# EXPOSE 8080

# Command to run the service
CMD ["./user_service"]