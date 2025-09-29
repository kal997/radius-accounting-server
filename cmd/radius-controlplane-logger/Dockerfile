# ---------- Build stage ----------
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build the binary for radius-controlplane-logger service
RUN go build -o /radius-controlplane-logger ./cmd/radius-controlplane-logger

# ---------- Run stage ----------
FROM alpine:latest

WORKDIR /app
COPY --from=builder /radius-controlplane-logger .


# Run binary
CMD ["./radius-controlplane-logger"]
