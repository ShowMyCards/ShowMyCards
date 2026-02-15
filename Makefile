.PHONY: all build build-backend build-frontend dev dev-backend dev-frontend test test-backend test-frontend types clean docker docker-build docker-up docker-down install release

VERSION ?= dev

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
	cd backend && go build -ldflags "-X backend/version.Version=$(VERSION)" -o bin/server .

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

# Release: tag and push a version
release:
	@test -n "$(VERSION)" || (echo "Usage: make release VERSION=1.2.3" && exit 1)
	@test "$(VERSION)" != "dev" || (echo "VERSION must be a semver, not 'dev'" && exit 1)
	@echo "Switching to main and pulling latest..."
	git checkout main
	git pull origin main
	@if [ -n "$$(git status --porcelain)" ]; then echo "Error: working directory is not clean" && exit 1; fi
	@if git rev-parse "v$(VERSION)" >/dev/null 2>&1; then echo "Error: tag v$(VERSION) already exists" && exit 1; fi
	git tag -a "v$(VERSION)" -m "Release v$(VERSION)"
	git push origin "v$(VERSION)"
	@echo "Released v$(VERSION)"

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
	@echo "  build-backend    Build Go backend (use VERSION=x.y.z to set version)"
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
	@echo "  docker-build     Build Docker image"
	@echo "  docker-up        Start container"
	@echo "  docker-down      Stop container"
	@echo ""
	@echo "Release:"
	@echo "  release          Tag and push a release (VERSION=x.y.z required)"
