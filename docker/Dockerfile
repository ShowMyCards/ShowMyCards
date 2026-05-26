# Combined Dockerfile
# Builds both backend and frontend into a single image

# ============================================
# Stage 1: Build Go backend
# ============================================
# Base image pinned to digest as defence against tag hijacks on Docker Hub
# (tags are mutable). Renovate's `docker` manager keeps the digest current
# and the version tag is retained as a human-readable hint.
FROM golang:1.26-alpine@sha256:91eda9776261207ea25fd06b5b7fed8d397dd2c0a283e77f2ab6e91bfa71079d AS backend-builder

WORKDIR /app

RUN apk add --no-cache gcc musl-dev

COPY backend/go.mod backend/go.sum ./
RUN go mod download

ARG VERSION=dev

COPY backend/ .
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags "-X backend/version.Version=${VERSION}" -o /server .

# ============================================
# Stage 2: Build SvelteKit frontend
# ============================================
FROM oven/bun:1-alpine@sha256:5acc90a93e91ff07bf72aa90a7c9f0fa189765aec90b47bdbf2152d2196383c0 AS frontend-builder

WORKDIR /app

COPY frontend/package.json frontend/bun.lock ./
RUN bun install --frozen-lockfile

COPY frontend/ .
RUN bun run build

# ============================================
# Stage 3: Runtime image
# ============================================
FROM alpine:3.23@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    sqlite \
    nodejs \
    npm \
    supervisor

# Copy backend binary
COPY --from=backend-builder /server ./backend/server

# Copy frontend build
COPY --from=frontend-builder /app/build ./frontend/build
COPY --from=frontend-builder /app/package.json ./frontend/

# Install frontend production dependencies
WORKDIR /app/frontend
RUN npm install --omit=dev --legacy-peer-deps
WORKDIR /app

# Create data directory for SQLite
RUN mkdir -p /app/data

# Copy supervisord config
COPY docker/supervisord.conf /etc/supervisor/conf.d/supervisord.conf

# Expose ports
EXPOSE 3000 3001

# Environment variables
ENV BACKEND_PORT=3000
ENV FRONTEND_PORT=3001
ENV ORIGIN=http://localhost:3001
# Writable data lives at /app/data (the volume mount point); anchor with absolute
# paths so the backend doesn't fall back to a cwd-relative ./data inside the
# container layer when supervisord sets directory=/app/backend.
ENV DATABASE_PATH=/app/data/database.db
ENV DATA_DIR=/app/data

# Run with supervisord
CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
