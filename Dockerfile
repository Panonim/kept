#########################
# Frontend build stage
#########################
FROM node:20-alpine AS frontend-builder
WORKDIR /src/frontend

# Copy package files and install
COPY frontend/package*.json ./
RUN if [ -f package-lock.json ]; then \
      npm ci --prefer-offline --no-audit --progress=false; \
    else \
      npm install --no-audit --progress=false; \
    fi

# Copy frontend sources and build
COPY frontend/ .
RUN npm run build

#########################
# Backend build stage
#########################
FROM golang:1.21-alpine AS backend-builder
WORKDIR /src/backend

# Install build deps for SQLite
RUN apk add --no-cache gcc musl-dev sqlite-dev

# Copy module files and download deps (cacheable)
COPY backend/go.mod backend/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# Copy backend sources and build
COPY backend/ .
RUN go mod tidy
ENV GOCACHE=/root/.cache/go-build
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=1 GOOS=linux go build -trimpath -ldflags "-s -w" -o /usr/local/bin/main .

#########################
# Final image
#########################
FROM alpine:latest
RUN apk add --no-cache nginx bash ca-certificates sqlite-libs gettext

# Copy frontend built files into nginx html dir
COPY --from=frontend-builder /src/frontend/dist /usr/share/nginx/html

# Copy backend binary
COPY --from=backend-builder /usr/local/bin/main /usr/local/bin/main

# Add nginx config template
RUN rm -rf /etc/nginx/http.d/default.conf
COPY nginx.conf /etc/nginx/http.d/default.conf.template
RUN mkdir -p /root/data /var/run/nginx

EXPOSE 80 3000

# Run backend in background and nginx in foreground so container stays up.
CMD ["sh", "-c", "export PORT=${PORT:-3000} && export FRONTEND_PORT=${FRONTEND_PORT:-80} && envsubst '${PORT} ${FRONTEND_PORT}' < /etc/nginx/http.d/default.conf.template > /etc/nginx/http.d/default.conf && /usr/local/bin/main & nginx -g 'daemon off;'"]
