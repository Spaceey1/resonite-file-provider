# Build stage
FROM golang:1.24 AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o resonite-file-provider . 
# -buildvcs=false
# Final stage
FROM alpine:latest

WORKDIR /app

# Install dependencies
# RUN apk --no-cache add ca-certificates tzdata

# Copy binary from build stage
COPY --from=builder /app/resonite-file-provider .
COPY --from=builder /app/data ./data

# Create ResoniteFilehost directory since it's expected by the system
# RUN mkdir -p ./ResoniteFilehost
# RUN touch ./ResoniteFilehost/placeholder.txt

# Create required directories
RUN mkdir -p ./assets
RUN mkdir -p ./certs

# Expose ports
EXPOSE 5819
# EXPOSE 8080

CMD ["./resonite-file-provider"]
