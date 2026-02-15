# Project Overview

This is a monorepo containing a self-hosted / locally-run web application with
a Go backend API and a Svelte frontend.

- **Backend** (`/backend`): Go 1.25, Fiber v3 API. Uses SQLite for storage and
  the Scryfall API for card data. Serves JSON responses for the frontend.
- **Frontend** (`/frontend`): Svelte 5 with SvelteKit. Consumes the backend API
  and renders the UI. Uses Bun as its package manager.
- **Website** (`/website`): Astro marketing site, deployed separately. Not part
  of the main application.

The application is intended to be self-hosted or run locally. There is
intentionally no authentication, authorisation, or rate limiting.

---

## Architecture

```
┌─────────────┐       JSON        ┌─────────────┐
│   Svelte 5  │ ◄───────────────► │  Go Fiber   │
│  SvelteKit  │                   │   v3 API    │
│  Frontend   │                   │   Backend   │
└─────────────┘                   └──────┬──────┘
                                         │
                                    ┌────┴────┐
                                    │         │
                                 SQLite    Scryfall
                                           API
```

The frontend and backend are independent deployable units that communicate
over HTTP/JSON. Changes to one should not require changes to the other unless
the API contract changes.

---

## Development

The `Makefile` in the project root is the primary interface for all development
tasks. Run `make help` for a full list of targets.

### Running the Application

```bash
# Install dependencies (Go modules + Bun)
make install

# Run backend and frontend in separate terminals
make dev-backend    # Go server on port 3000
make dev-frontend   # SvelteKit on port 5173
```

### Environment Variables

All configuration is via environment variables. The frontend reads
`VITE_BACKEND_URL` (defaults to `http://localhost:3000`). The backend reads
`PORT` (defaults to `3000`). For Docker deployments, environment variables are
set in `docker-compose.yml`. Never hardcode connection strings, API URLs, ports,
or feature flags.

### Docker

A combined Docker image runs both backend and frontend via supervisord:

```bash
make docker-build   # Build image
make docker-up      # Start container (ports 3000 + 3001)
make docker-down    # Stop container
```

---

## Monorepo Conventions

### Directory-Level CLAUDE.md Files

Each directory has its own `CLAUDE.md` with context specific to that part of
the codebase:

- `/backend/CLAUDE.md` — Go idioms, Fiber v3 patterns, API endpoints, domain
  model, test/lint commands.
- `/frontend/CLAUDE.md` — Svelte 5 runes, SvelteKit conventions, component
  library, design patterns, test/lint commands.

When working in a subdirectory, follow both this file and the local `CLAUDE.md`.
If they conflict, the local file takes precedence for domain-specific guidance;
this file takes precedence for cross-cutting concerns.

### Code Reviews

Code reviews follow a structured, multi-pass process defined in
`REVIEW_AGENT.md`. Each section of the codebase has its own review standards:

- `/backend/BACKEND_REVIEW_STANDARDS.md` — Go review rubric.
- `/frontend/FRONTEND_REVIEW_STANDARDS.md` — Svelte review rubric.

When performing reviews, use the review agent instructions and the appropriate
standards file. Do not invent criteria outside the standards.

### Shared Principles

These apply to both codebases:

1. **Twelve-factor app.** Configuration from environment variables. Logs to
   stdout. Backing services as attached resources. Graceful shutdown. No
   environment-specific code paths.

2. **No hardcoded values.** API URLs, ports, database connection strings,
   external service endpoints, and feature flags come from config — never from
   source code.

3. **Explicit dependencies.** All dependencies are declared in `go.mod`
   (backend) or `package.json` (frontend). No implicit reliance on globally
   installed tools.

4. **Consistent error handling.** Both codebases handle errors explicitly.
   The backend wraps errors with context. The frontend handles loading, error,
   and empty states for every data-fetching operation.

5. **Type safety.** The backend uses Go's type system. The frontend uses
   TypeScript in strict mode. No `any` without documented justification.

6. **Test what matters.** Both codebases should have test coverage for their
   public API surfaces and critical paths. Tests assert on behaviour, not
   implementation details.

### API Contract & Type Generation

The backend serves JSON responses consumed by the frontend. TypeScript types
are auto-generated from Go structs via [Tygo](https://github.com/gzuidhof/tygo).
Run `make types` to regenerate after changing backend models or API response
types.

Generated files (do not edit manually):
- `frontend/src/lib/types/models.ts` — from `backend/models/*.go`
- `frontend/src/lib/types/api.ts` — from `backend/api/*.go`

When modifying API endpoints:

- Update the backend handler and response types.
- Run `make types` to regenerate the frontend TypeScript types.
- Both sides must agree on the response shape. If a field is added, removed,
  or renamed, both codebases need updating.

### Commit & PR Practices

- Commits that touch both `/backend` and `/frontend` should clearly describe
  what changed in each.
- If a change requires coordinated updates (e.g. a new API field), make the
  backend change first, then the frontend change. The frontend should never
  depend on a field that the backend doesn't yet serve.

---

## Tooling Quick Reference

| Task | Backend | Frontend |
|------|---------|----------|
| Run dev server | `make dev-backend` | `make dev-frontend` |
| Run tests | `make test-backend` | `make test-frontend` |
| Run linters | `cd backend && golangci-lint run ./...` | `cd frontend && bunx svelte-check && bunx eslint .` |
| Type check | (implicit in Go) | `cd frontend && bunx svelte-check` |
| Format | `cd backend && gofmt -w .` | `cd frontend && bunx prettier --write .` |
| Generate types | `make types` | — |

---

## Things Claude Should Know

- This is a self-hosted / local application. Do not suggest cloud-specific
  services, managed databases, or SaaS dependencies unless explicitly asked.
- There is no authentication or authorisation by design. Do not add it or
  suggest adding it.
- There is no rate limiting by design. Do not add it or suggest adding it.
- The backend and frontend are separate concerns. Do not mix Go and Svelte
  guidance. Use the appropriate directory-level CLAUDE.md for each.
- The frontend uses **Bun**, not npm. Use `bun` / `bunx` for all frontend
  commands.
- When making changes, prefer the simplest approach that works. Do not
  introduce abstractions, patterns, or dependencies without a concrete
  present need.
