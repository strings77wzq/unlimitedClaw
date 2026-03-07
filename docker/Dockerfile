# ============================================================
# Stage 1: Build the golem binary
# ============================================================
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -o /build/golem ./cmd/golem

# ============================================================
# Stage 2: Minimal runtime image
# ============================================================
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata wget

# Copy binary from builder
COPY --from=builder /build/golem /usr/local/bin/golem

# Create non-root user
RUN adduser -D -g '' appuser

# Switch to non-root user
USER appuser

EXPOSE 18790

# Health check
HEALTHCHECK --interval=30s --timeout=3s \
  CMD wget -qO- http://localhost:18790/health || exit 1

ENTRYPOINT ["/usr/local/bin/golem"]
CMD ["gateway"]
