# Backend Dockerfile
# syntax=docker/dockerfile:1.4
FROM golang:1.24 AS builder

WORKDIR /app

# Install build dependencies for SQLite/SQLCipher on Debian-based image
RUN apt-get update && apt-get install -y --no-install-recommends \
	git \
	wget \
	gcc \
	build-essential \
	libssl-dev \
	tcl \
	pkg-config \
	ca-certificates \
	autoconf \
	automake \
	libtool \
 && rm -rf /var/lib/apt/lists/*

# Build and install SQLCipher from source so Go sqlite3 can link against it
RUN set -eux; \
	git clone --depth 1 --branch v4.5.4 https://github.com/sqlcipher/sqlcipher.git /tmp/sqlcipher; \
	cd /tmp/sqlcipher; \
	./configure --enable-tempstore=yes CFLAGS="-DSQLITE_HAS_CODEC -fPIC" LDFLAGS="-lcrypto"; \
	make && make install; \
	rm -rf /tmp/sqlcipher

# Copy only module files first to leverage Docker layer caching for dependencies
COPY go.mod ./

# Download modules (cached unless go.mod changes)
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# Copy rest of the source
COPY . .

# Ensure go.sum is populated and tidy the module (faster since modules were downloaded)
RUN go mod tidy

# Build the application with CGO enabled for SQLite
# Use build cache mounts for Go build cache and module cache to speed repeated builds
ENV GOCACHE=/root/.cache/go-build
RUN --mount=type=cache,target=/root/.cache/go-build \
	--mount=type=cache,target=/go/pkg/mod \
	CGO_ENABLED=1 GOOS=linux go build -trimpath -ldflags "-s -w" -o main .

# Final stage
FROM debian:bookworm-slim

# Install runtime dependencies needed by the healthcheck (wget)
RUN apt-get update && apt-get install -y --no-install-recommends \
	ca-certificates \
	libssl3 \
	wget \
 && rm -rf /var/lib/apt/lists/*

WORKDIR /root/

# Copy SQLCipher shared library files from builder so runtime can load them
COPY --from=builder /usr/local/lib/libsqlcipher* /usr/local/lib/
RUN ldconfig || true

# Copy the binary from builder
COPY --from=builder /app/main .

# Create data directory
RUN mkdir -p /root/data

EXPOSE 3000

CMD ["./main"]
