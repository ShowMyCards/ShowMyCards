.PHONY: all build build-backend build-frontend dev dev-backend dev-frontend test test-backend test-frontend types clean docker docker-build docker-up docker-down install manual-release repo-settings

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

# Manual release: tag and push a version directly, bypassing release-please.
#
# This is the escape hatch for cases release-please can't handle — typically
# a hotfix tagged from a non-default branch, or a re-tag after a botched
# release. For normal releases, merge the open "chore: release X.Y.Z" PR on
# main and let release-please cut the tag.
manual-release:
	@test -n "$(VERSION)" || (echo "Usage: make manual-release VERSION=1.2.3" && exit 1)
	@test "$(VERSION)" != "dev" || (echo "VERSION must be a semver, not 'dev'" && exit 1)
	@echo "WARNING: This bypasses release-please and will not update CHANGELOG.md."
	@echo "         Only use for hotfixes or out-of-band releases. For normal"
	@echo "         releases, merge the open 'chore: release' PR on main instead."
	@echo ""
	@if [ -n "$$(git status --porcelain)" ]; then echo "Error: working directory is not clean" && exit 1; fi
	@if git rev-parse "v$(VERSION)" >/dev/null 2>&1; then echo "Error: tag v$(VERSION) already exists" && exit 1; fi
	git tag -a "v$(VERSION)" -m "Release v$(VERSION)"
	git push origin "v$(VERSION)"
	@echo "Released v$(VERSION)"

# Apply repository settings and branch protection to GitHub.
# The shell script is the canonical record — any change to repo settings
# should be made there, not in the GitHub UI, so this file stays in sync
# with reality.
repo-settings:
	@command -v gh >/dev/null || (echo "Error: gh CLI is required" && exit 1)
	@gh auth status >/dev/null 2>&1 || (echo "Error: run 'gh auth login' first" && exit 1)
	./scripts/apply-repo-settings.sh

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
	@echo "  Normal releases are driven by release-please — merge the open"
	@echo "  'chore: release X.Y.Z' PR on main and the tag is cut automatically."
	@echo "  manual-release   Tag and push a release out-of-band, bypassing"
	@echo "                   release-please (VERSION=x.y.z required)."
	@echo ""
	@echo "Repo administration (maintainers only):"
	@echo "  repo-settings    Apply GitHub repo settings + branch protection"
