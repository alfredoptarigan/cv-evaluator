FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
  git \
  ca-certificates \
  tzdata

WORKDIR /app

# Copy dependency files first (for better caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -ldflags="-w -s" \
  -o /app/api \
  ./cmd/api/main.go

# Build ingestion script
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -ldflags="-w -s" \
  -o /app/ingest \
  ./scripts/ingest_documents.go

# ============================================
# Final stage - minimal runtime image
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add \
  ca-certificates \
  tzdata \
  curl

# Create non-root user
RUN addgroup -g 1000 appuser && \
  adduser -D -u 1000 -G appuser appuser

WORKDIR /app

# Copy binaries from builder
COPY --from=builder /app/api /app/api
COPY --from=builder /app/ingest /app/ingest

# Create necessary directories
RUN mkdir -p /app/uploads /app/reference_docs /app/logs && \
  chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 3000

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:3000/api/v1/health || exit 1

# Run the application
CMD ["/app/api"]