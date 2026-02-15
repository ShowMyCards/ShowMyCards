.PHONY: all build build-backend build-frontend dev dev-backend dev-frontend test test-backend test-frontend types clean docker docker-build docker-up docker-down install

# Default target
all: build

# Install dependencies
install:
	cd backend && go mod download
	cd frontend && bun install
	cd website && bun install

# Generate TypeScript types from Go models
types:
	cd backend && go run github.com/gzuidhof/tygo@latest generate

# Build targets
build: build-backend build-frontend

build-backend:
	cd backend && go build -o bin/server .

build-frontend:
	cd frontend && bun run build

# Development servers
dev-backend:
	cd backend && go run .

dev-frontend:
	cd frontend && bun run dev

dev-website:
	cd website && bun run dev

# Run both backend and frontend (requires terminal multiplexer or separate terminals)
dev:
	@echo "Run 'make dev-backend' and 'make dev-frontend' in separate terminals"

# Test targets
test: test-backend test-frontend

test-backend:
	cd backend && go test ./...

test-frontend:
	cd frontend && bun run test

# Lint and format
lint:
	cd frontend && bun run lint

format:
	cd frontend && bun run format

# Clean build artifacts
clean:
	rm -rf backend/bin
	rm -rf frontend/build
	rm -rf frontend/.svelte-kit
	rm -rf frontend/node_modules
	rm -rf website/node_modules
	rm -rf website/.astro

# Docker targets
docker-build:
	docker build -t showmycards:latest -f docker/Dockerfile .

docker-up:
	docker compose up -d

docker-down:
	docker compose down

# Help
help:
	@echo "ShowMyCards Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  install          Install all dependencies"
	@echo "  types            Generate TypeScript types from Go models"
	@echo "  build            Build backend and frontend"
	@echo "  build-backend    Build Go backend"
	@echo "  build-frontend   Build SvelteKit frontend"
	@echo "  dev-backend      Run backend dev server"
	@echo "  dev-frontend     Run frontend dev server"
	@echo "  dev-website      Run website dev server"
	@echo "  test             Run all tests"
	@echo "  test-backend     Run Go tests"
	@echo "  test-frontend    Run frontend tests"
	@echo "  lint             Lint frontend code"
	@echo "  format           Format frontend code"
	@echo "  clean            Remove build artifacts"
	@echo ""
	@echo "Docker:"
	@echo "  docker-build           Build Docker image"
	@echo "  docker-up              Start container"
	@echo "  docker-down            Stop container"
