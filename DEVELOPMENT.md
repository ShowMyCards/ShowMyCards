# Development Guide

## Prerequisites

- Go 1.25+
- Node.js 22+ (or Bun)
- Docker (optional, for containerized development)

## Project Structure

```
ShowMyCards/
├── backend/           # Go API server
│   ├── api/           # HTTP handlers
│   ├── database/      # Database connection and migrations
│   ├── models/        # Domain models (source of truth for types)
│   ├── rules/         # Rule evaluation engine
│   ├── scryfall/      # Scryfall API client
│   ├── server/        # Server setup and routing
│   ├── services/      # Business logic
│   ├── utils/         # Utility functions
│   ├── docs/          # Swagger documentation
│   └── tygo.yaml      # TypeScript type generation config
├── frontend/          # SvelteKit web application
│   └── src/lib/types/ # Auto-generated TypeScript types
├── website/           # Astro marketing site (deployed separately)
├── docker/            # Dockerfiles and configs
├── Makefile           # Build commands
└── docker-compose.yml # Default Docker Compose config
```

## Getting Started

### Install Dependencies

```bash
make install
```

This will:
- Download Go modules
- Install frontend dependencies with Bun
- Install website dependencies with Bun

### Run Development Servers

Run the backend and frontend in separate terminals:

```bash
# Terminal 1 - Backend (port 3000)
make dev-backend

# Terminal 2 - Frontend (port 5173)
make dev-frontend
```

Or for the marketing website:
```bash
make dev-website
```

## Type Generation

TypeScript types are automatically generated from Go structs using [Tygo](https://github.com/gzuidhof/tygo).

### Generate Types

```bash
make types
```

This generates:
- `frontend/src/lib/types/models.ts` - From `backend/models/`
- `frontend/src/lib/types/api.ts` - From `backend/api/`

### Adding Exported Types

Add the `// tygo:export` comment above any struct to export it to TypeScript:

```go
// tygo:export
type MyNewType struct {
    ID   uint   `json:"id"`
    Name string `json:"name"`
}
```

### Type Mappings

| Go Type | TypeScript Type |
|---------|-----------------|
| `string` | `string` |
| `int`, `uint`, `int64` | `number` |
| `bool` | `boolean` |
| `time.Time` | `string` (ISO 8601) |
| `*T` (pointer) | `T \| undefined` |
| `[]T` (slice) | `T[]` |

## Testing

```bash
# Run all tests
make test

# Backend tests only
make test-backend

# Frontend tests only
make test-frontend
```

## Building

```bash
# Build both backend and frontend
make build

# Backend only
make build-backend

# Frontend only
make build-frontend
```

## Docker

### Build Images

```bash
# Combined image (recommended)
make docker-build

# Separate images
make docker-build-backend
make docker-build-frontend
```

### Run with Docker Compose

```bash
# Combined container (default)
make docker-up
make docker-down

# Separate containers
make docker-up-separate
make docker-down-separate
```

### Docker Architecture

**Combined Image** (`docker/Dockerfile`):
- Single container with supervisord managing both services
- Backend API on port 3000, Frontend on port 3001
- Simpler deployment, single volume for data

**Separate Images** (`docker/Dockerfile.backend`, `docker/Dockerfile.frontend`):
- Independent containers for backend and frontend
- Better for scaling and independent deployments
- Frontend connects to backend via Docker networking

## Code Style

### Backend (Go)

- Standard Go formatting (`go fmt`)
- Follow existing patterns in the codebase

### Frontend (TypeScript/Svelte)

```bash
# Check formatting
make lint

# Auto-format
make format
```

## API Documentation

Swagger UI is available at `http://localhost:3000/swagger` when running the backend.

## Make Commands Reference

Run `make help` to see all available commands:

```
Development:
  install          Install all dependencies
  types            Generate TypeScript types from Go models
  dev-backend      Run backend dev server
  dev-frontend     Run frontend dev server
  dev-website      Run website dev server

Building:
  build            Build backend and frontend
  build-backend    Build Go backend
  build-frontend   Build SvelteKit frontend

Testing:
  test             Run all tests
  test-backend     Run Go tests
  test-frontend    Run frontend tests

Code Quality:
  lint             Lint frontend code
  format           Format frontend code
  clean            Remove build artifacts

Docker (combined image - default):
  docker-build           Build combined Docker image
  docker-up              Start combined container
  docker-down            Stop combined container

Docker (separate images):
  docker-build-backend   Build backend Docker image
  docker-build-frontend  Build frontend Docker image
  docker-up-separate     Start separate containers
  docker-down-separate   Stop separate containers
```
