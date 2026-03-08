# =============================================================================
# Stage 1: Builder
# =============================================================================
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Install build-time system dependencies
RUN apk add --no-cache git ca-certificates tzdata make

# Download Go modules separately for layer caching.
# go.sum must exist (run `go mod tidy` locally before building).
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy the full source tree
COPY . .

# Build arguments for embedding version metadata
ARG VERSION=dev
ARG BUILD_TIME=unknown

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags="-w -s -X 'main.Version=${VERSION}' -X 'main.BuildTime=${BUILD_TIME}'" \
    -o /build/api \
    ./cmd/api/

# =============================================================================
# Stage 2: Runner
# =============================================================================
FROM alpine:3.19 AS runner

# Install minimal runtime dependencies
RUN apk add --no-cache \
        ca-certificates \
        tzdata \
        curl \
    && update-ca-certificates

# Create a non-root system user (uid 1001) for security
RUN addgroup -g 1001 -S appgroup \
    && adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

# Copy the compiled binary from the builder stage
COPY --from=builder --chown=appuser:appgroup /build/api ./api

# Ensure the upload temp directory exists and is writable by the app user
RUN mkdir -p /tmp/uploads \
    && chown appuser:appgroup /tmp/uploads

# Drop privileges
USER appuser

EXPOSE 8080

# Health check: the /health endpoint must return HTTP 200 within 10 s.
HEALTHCHECK \
    --interval=30s \
    --timeout=10s \
    --start-period=10s \
    --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

ENTRYPOINT ["./api"]
