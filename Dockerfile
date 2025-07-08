# Stage 1: Build the application
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum files to download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the application. CGO_ENABLED=0 creates a static binary.
RUN CGO_ENABLED=0 GOOS=linux go build -o /spark-api ./cmd/api

# Stage 2: Create the final, small image
FROM alpine:latest

WORKDIR /

# Copy the built binary from the builder stage
COPY --from=builder /spark-api /spark-api

# Expose the port the app runs on
EXPOSE 8080

# Command to run the executable
CMD ["/spark-api"]