# ---------- Build stage ----------
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build the binary for radius-controlplane service
RUN go build -o /radius-controlplane ./cmd/radius-controlplane

# ---------- Run stage ----------
FROM alpine:latest

WORKDIR /app
COPY --from=builder /radius-controlplane .

# Expose port 1813 for serving accounting reqs
EXPOSE 1813/udp

# Run binary
CMD ["./radius-controlplane"]
