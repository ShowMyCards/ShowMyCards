# Contributing to ShowMyCards

Thanks for considering a contribution. This document covers the practical
things you need to know to get a change merged.

## Before You Start

- For anything beyond a small fix or typo, **open an issue first** to discuss
  the change. This avoids wasted work if the change does not fit the project's
  direction.
- This is a self-hosted, locally-run application. Features that assume a
  hosted/cloud deployment, add authentication, add **inbound** rate limiting
  to the backend API, or introduce SaaS dependencies are out of scope unless
  explicitly invited.
- **Outbound** rate limiting toward third-party APIs (such as Scryfall) is
  welcome and expected — we should be a good citizen of any service we call.

## Development Setup

See [DEVELOPMENT.md](DEVELOPMENT.md) for the full setup. The short version:

```bash
make install        # Go modules + Bun packages
make dev-backend    # Go API on :3000
make dev-frontend   # SvelteKit on :5173
```

The `Makefile` is the canonical entry point. Run `make help` for a full list
of targets.

## Making Changes

1. Fork the repository and create a branch from `main`. Name it something
   descriptive: `fix/...`, `feat/...`, `chore/...`, etc.
2. Make your change. Keep PRs focused — one logical change per PR.
3. Run the relevant checks locally before pushing:
   - Backend: `make test-backend` and `cd backend && go vet ./...`
   - Frontend: `cd frontend && bun run check && bun run lint && bunx vitest --run --project server`
4. If you changed any backend models or API response types, run `make types`
   to regenerate the frontend TypeScript types.
5. Push and open a pull request.

## Commit Messages

This project uses [Conventional Commits](https://www.conventionalcommits.org/)
to drive automated releases. Use one of these prefixes:

- `feat: ...` — a new feature (bumps the minor version)
- `fix: ...` — a bug fix (bumps the patch version)
- `feat!: ...` or `fix!: ...` — a breaking change (bumps the major version)
- `chore: ...`, `docs: ...`, `refactor: ...`, `test: ...`, `ci: ...` —
  housekeeping (no version bump)

Squash-and-merge is the only merge style available, so only the PR title
needs to follow this format. The PR title becomes the squash commit subject.

## Pull Request Review

All PRs require:

- A green CI run (tests, lint, type check, vulnerability scans).
- One approval from a maintainer.
- An up-to-date branch relative to `main`.

External contributor workflow runs will require manual approval from a
maintainer the first time. This is intentional — please be patient.

## What Not to Include in a PR

- Generated files (e.g. `frontend/src/lib/types/models.ts`,
  `frontend/src/lib/types/api.ts`) unless the change to the source warrants
  regenerating them. Run `make types` if you touched backend models.
- Formatting-only changes mixed with functional changes — split them.
- Dependency upgrades. Those are handled by Renovate, please don't bundle
  them with feature work.
- `package-lock.json`, `yarn.lock`, or other non-Bun lockfiles for the
  frontend or website (both use `bun.lock`).

## Reporting Security Issues

Please **do not** open a public issue for security vulnerabilities. See
[SECURITY.md](SECURITY.md) for the disclosure process.

## Repository Administration

The branch protection rules, merge settings, and Actions permissions for this
repository are managed as code in
[scripts/apply-repo-settings.sh](scripts/apply-repo-settings.sh). That script
is the **canonical record** of the repo's configuration — any change to
settings should be made there and applied via `make repo-settings`, not
through the GitHub web UI. If you spot a discrepancy between the script and
the live repo, that's a bug worth reporting.

Only maintainers need to run this; it requires admin access on the repo.
