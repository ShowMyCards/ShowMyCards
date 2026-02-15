# Combined Dockerfile
# Builds both backend and frontend into a single image

# ============================================
# Stage 1: Build Go backend
# ============================================
FROM golang:1.25-alpine AS backend-builder

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
FROM node:22-alpine AS frontend-builder

WORKDIR /app

RUN npm install -g bun

COPY frontend/package.json frontend/bun.lock ./
RUN bun install --frozen-lockfile

COPY frontend/ .
RUN bun run build

# ============================================
# Stage 3: Runtime image
# ============================================
FROM alpine:3.23

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

# Run with supervisord
CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
