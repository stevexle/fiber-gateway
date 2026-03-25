# ─── STEP 1: Builder ───
FROM golang:1.26.1-alpine AS builder

# Add essential build tools
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Dependency management (cached)
COPY go.mod go.sum ./
RUN go mod download

# Build application
COPY . .
# -s -w: Strips symbol table and debug information for smallest binary size
# CGO_ENABLED=0: Disables dynamic linking for a truly standalone binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -a -installsuffix cgo -o fiber-gateway .

# ─── STEP 2: Optimized Production Runtime ───
FROM alpine:3.19

# Security: Set up a non-root user for the gateway
RUN addgroup -S gateway && adduser -S gateway -G gateway

# Robustness: Install common runtime essentials
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Ensure log directory exists and is owned by the app user
RUN mkdir -p /app/logs && chown -R gateway:gateway /app

# Runtime configuration defaults
ENV PORT=8080
EXPOSE ${PORT}

# Copy the binary and static configuration
COPY --from=builder /app/fiber-gateway /app/fiber-gateway
COPY --from=builder /app/routes.json /app/routes.json

# Security: Run as a non-root user
USER gateway

# Healthcheck for orchestration (Robustness)
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget -nv -t1 --spider http://localhost:${PORT}/api/v1/health || exit 1

ENTRYPOINT ["/app/fiber-gateway"]