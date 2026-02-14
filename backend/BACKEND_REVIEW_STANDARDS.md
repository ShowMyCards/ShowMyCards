# Code Review Standards

> **Purpose:** Define a finite, repeatable rubric for AI-assisted code reviews.
> Reviews check _only_ what is listed here. If an issue does not fall into a
> category below, it is out of scope and must not be flagged.

---

## How to Use This Document

When performing a code review, work through each section in order. For every
finding, assign exactly one severity from the table below. Do not invent new
severities or categories.

| Severity | Meaning | Action |
|----------|---------|--------|
| **MUST FIX** | Bugs, security holes, data loss risk, twelve-factor violations that break portability | Block merge |
| **SHOULD FIX** | Meaningful improvements to maintainability, testability, or idiom conformance | Fix in this PR or create a tracked issue |
| **CONSIDER** | Minor suggestions that are genuinely better but where the current code is acceptable | Author's discretion — no follow-up required |

Present findings as a structured list grouped by severity. Each finding must
reference a specific file and line range, state what the problem is, explain
_why_ it matters, and suggest a concrete fix.

---

## 1 · Correctness & Safety

These are bugs or near-bugs. Always flag.

- **Nil/zero-value dereferences.** Especially watch for unchecked type assertions,
  nil map writes, and nil slices passed to functions expecting non-nil.
- **Unclosed resources.** Database rows, response bodies, file handles, and
  anything implementing `io.Closer` must be closed — typically via `defer` —
  on every code path including error paths.
- **Goroutine leaks.** Any goroutine launched must have a clear shutdown signal
  (context cancellation, channel close, `sync.WaitGroup`). Goroutines that can
  block forever without a shutdown path are MUST FIX.
- **Race conditions.** Shared mutable state accessed from multiple goroutines
  without synchronisation (`sync.Mutex`, `sync.RWMutex`, channels, or atomics).
  If unsure, the code should be safe to run under `go test -race`.
- **Error swallowing.** Every returned `error` must be checked. Errors must not
  be silently discarded with `_`. If an error truly cannot be handled, it must
  be logged with sufficient context.
- **Fiber context safety.** Values obtained from `fiber.Ctx` (params, body,
  headers) are only valid within the handler. Any value that escapes the handler
  must be copied (see Fiber v3 docs on immutability).

---

## 2 · Error Handling

Go's explicit error handling is a strength when done consistently.

- **Wrap with context.** Use `fmt.Errorf("doing X: %w", err)` so that callers
  up the stack can understand _where_ and _why_ an error occurred. Bare
  `return err` without added context is SHOULD FIX when the function does
  anything non-trivial.
- **Sentinel errors and custom types.** Where callers need to branch on error
  kind, define sentinel errors (`var ErrNotFound = errors.New(...)`) or custom
  error types. Do not compare error strings.
- **Don't panic.** `panic` is reserved for genuinely unrecoverable programmer
  errors (e.g. impossible state after exhaustive switch). Never panic on bad
  user input, failed I/O, or external API failures.
- **Fiber error handling.** Use Fiber's `fiber.NewError(statusCode, message)` or
  a custom error handler on the app to produce consistent JSON error responses.
  Handlers must not write mixed response formats.

---

## 3 · Twelve-Factor App Compliance

The application must remain portable, environment-agnostic, and easy to deploy.
Flag violations of these principles as MUST FIX unless noted otherwise.

### 3.1 — Config (Factor III)

- **All configuration must come from environment variables.** No hardcoded
  connection strings, API URLs, ports, feature flags, or timeouts. No `.env`
  files committed to the repo (a `.env.example` with placeholder values is fine).
- **Parse config once at startup** into a typed struct. The rest of the
  application receives config via dependency injection — never by reading
  `os.Getenv` deep inside business logic.
- **Validate config eagerly.** Missing or invalid required config should cause a
  fast, clear startup failure — not a runtime nil-pointer ten minutes later.

### 3.2 — Backing Services (Factor IV)

- **Treat the database and external API as attached resources.** Connection
  details must be config-driven. Swapping from a local SQLite to a remote
  Postgres (or from one API host to another) must require only a config change,
  never a code change.
- **No implicit singletons.** Database pools, HTTP clients for the external API,
  and similar resources must be created explicitly and passed via dependency
  injection — not accessed through package-level globals.

### 3.3 — Port Binding (Factor VII)

- **The app exports HTTP via port binding.** The listen address/port must come
  from config (`PORT`, `LISTEN_ADDR`, or equivalent). Do not hardcode `:3000`.

### 3.4 — Disposability (Factor IX)

- **Fast startup.** The app should be ready to serve within seconds. Heavy
  initialisation (like pre-warming caches) should be non-blocking where possible.
- **Graceful shutdown.** The app must handle `SIGTERM`/`SIGINT`, stop accepting
  new requests, drain in-flight requests within a configurable timeout, close
  database connections, and exit cleanly. Fiber v3's `app.ShutdownWithContext`
  supports this.

### 3.5 — Dev/Prod Parity (Factor X)

- **No environment-specific code paths** like `if os.Getenv("ENV") == "dev"`.
  Behaviour differences should be driven by config values, not environment
  names. Logging level, for example, is a config knob — not an if/else on
  the environment string.

### 3.6 — Logs (Factor XI)

- **Logs are event streams written to stdout.** Do not write to files, do not
  manage log rotation. Use structured logging (`log/slog` from the standard
  library). The log level should be configurable via environment variable.

### 3.7 — Admin Processes (Factor XII)

- **One-off tasks** (migrations, seed scripts) should be runnable as standalone
  commands or subcommands of the same binary — not hidden functions called via
  a secret API endpoint.

---

## 4 · Idiomatic Go

Flag deviations from accepted Go idiom. These make the codebase harder to read
for any Go developer who inherits it.

- **Accept interfaces, return structs.** Function parameters should depend on
  the narrowest interface they need. Return concrete types so callers can access
  the full API.
- **Use standard naming.** `MixedCaps` / `mixedCaps`, not underscores. Acronyms
  are all-caps (`HTTP`, `ID`, `URL`). Receiver names are short (1–2 letters),
  consistent across methods, and never `this` or `self`.
- **Keep interfaces small.** Prefer one- or two-method interfaces defined by the
  consumer, not the implementer. Large "kitchen sink" interfaces are SHOULD FIX.
- **Use `context.Context` correctly.** It is always the first parameter, named
  `ctx`. Never store it in a struct. Propagate it through the call chain to
  enable cancellation and deadlines, especially for database queries and external
  API calls.
- **Struct initialisation.** Use named fields (`Foo{Bar: "x"}`), not positional.
  Avoid `new(T)` when a literal is clearer.
- **Effective use of `defer`.** Resource cleanup should use `defer` immediately
  after acquisition. Watch for `defer` inside loops (accumulates deferred calls).
- **Package naming.** Packages are lowercase, single-word where possible. No
  `utils`, `helpers`, `common`, or `models` grab-bag packages. Package names
  should describe what the package _does_, not what it _contains_.
- **Don't stutter.** A type `Client` in package `http` is `http.Client` — not
  `http.HTTPClient`. Avoid repeating the package name in exported identifiers.
- **Exported identifiers need doc comments.** Every exported function, type, and
  constant should have a comment starting with its name. Unexported identifiers
  should be documented when their purpose is non-obvious.

---

## 5 · Structure, Decomposition & Duplication

This is where monolithic code gets flagged — but only when decomposition
provides a real, tangible benefit.

### When to Flag

- **Functions exceeding ~60 lines.** This is a guideline, not a hard rule. A
  long function that does one thing linearly (e.g. a sequential pipeline) may be
  fine. A 40-line function with nested conditionals and multiple concerns is not.
- **Duplicated logic appearing 3+ times.** Two instances of similar code may be
  coincidence. Three is a pattern. Extract it.
- **Mixed abstraction levels.** A handler that parses input, calls the database,
  applies business rules, formats output, and writes the response is doing too
  much. Separate concerns into layers:
  - **Handlers** — parse request, call service, write response.
  - **Services** — business logic, orchestration. No knowledge of HTTP.
  - **Repositories / Clients** — data access and external API calls. No business logic.
- **God structs.** A struct with 10+ fields or methods spanning multiple concerns
  should be decomposed.

### When NOT to Flag

- **Extracting a function used exactly once** just to make a function shorter.
  If it doesn't clarify, don't extract.
- **Premature abstraction.** Do not suggest interfaces, generics, or strategy
  patterns for code that currently has one concrete implementation and no
  realistic prospect of a second.
- **Cosmetic rearrangement.** Moving code between files without changing the
  public API or reducing coupling is not an improvement.

---

## 6 · Coupling & Testability

Low coupling enables testing, refactoring, and replacement of components.

- **Depend on behaviour, not implementation.** Functions should accept interfaces
  (or function types) rather than concrete structs when the concrete type is an
  external dependency (database, API client, clock, etc.).
- **No `init()` with side effects.** `init()` should only register things (like
  database drivers). It must not open connections, read config, or start
  goroutines. This makes testing and startup ordering unpredictable.
- **Constructor injection.** Components receive their dependencies through
  constructor functions (`func NewService(repo Repository, client APIClient)
  *Service`). This makes dependencies explicit and testing straightforward.
- **Table-driven tests.** Prefer `[]struct{ name string; ... }` test tables for
  functions with multiple input/output scenarios. Subtests with `t.Run(tc.name,
  ...)` give clear failure messages.
- **Test the behaviour, not the implementation.** Tests should assert on
  outputs and observable side effects, not on which internal methods were called
  in which order. Avoid testing private methods directly.
- **Use `testing/synctest` for concurrent code.** Go 1.25 stabilised this
  package for testing goroutine-based logic with a virtualised clock. Prefer it
  over flaky `time.Sleep`-based approaches.
- **External API client must be mockable.** The HTTP client used to call the
  external API should be behind an interface so tests can substitute a fake
  without hitting the network. Consider defining the interface at the consumer
  site (in the service package, not the client package).

---

## 7 · Database Access

- **Use parameterised queries.** No string concatenation or `fmt.Sprintf` for
  SQL. Even though this is a local app without auth, injection bugs are still
  correctness bugs.
- **Close rows.** `sql.Rows` must always be closed, even on error. Use
  `defer rows.Close()` immediately after the query call.
- **Manage connections via pool.** Use `sql.DB` (which is a pool) or your ORM's
  equivalent. Do not open/close connections per request.
- **Migrations are versioned and repeatable.** Schema changes must be tracked in
  version-controlled migration files. Running them must be idempotent.
- **Context propagation.** All database calls should accept and use
  `context.Context` to honour request cancellation and timeouts. Use `QueryContext`,
  `ExecContext`, etc.

---

## 8 · External API Client

- **Centralise HTTP client configuration.** Timeouts, retry logic, and base URL
  are set once when the client is constructed — not scattered across individual
  calls.
- **Set sensible timeouts.** An `http.Client` with no timeout will block
  indefinitely. Always configure `Timeout` or use context deadlines.
- **Handle non-2xx responses explicitly.** A `nil` error from `http.Client.Do`
  only means the request completed — not that it succeeded. Always check
  `resp.StatusCode`.
- **Close response bodies.** `resp.Body.Close()` must be deferred after checking
  for a non-nil response, on every code path.
- **Structured error mapping.** Map external API errors to your own domain error
  types rather than surfacing raw HTTP status codes or third-party error strings
  to the frontend.

---

## 9 · JSON API Responses

- **Consistent response envelope.** All endpoints should return a consistent
  JSON shape. Error responses and success responses should be predictable for
  the frontend consumer.
- **Use appropriate status codes.** `200` for success, `201` for creation,
  `404` for not found, `500` for unexpected errors, etc. Don't return `200` with
  an error body.
- **Don't leak internals.** Stack traces, raw SQL errors, and file paths must
  never appear in production responses. Log them server-side; return a generic
  message to the client.
- **Use struct tags for serialisation.** All response structs should have
  explicit `json:"field_name"` tags. Prefer `camelCase` for JSON field names to
  align with JavaScript/frontend conventions.
- **Omit empty fields intentionally.** Use `omitempty` only when the absence of a
  field is semantically meaningful to the consumer. Don't sprinkle it everywhere.

---

## 10 · Linter & Tooling Baseline

These items are handled by automated tooling and **must not be flagged in review**.
Ensure the following are configured in CI; defer to their output.

- `gofmt` / `goimports` — formatting.
- `go vet` — static analysis for common mistakes.
- `golangci-lint` — with at minimum: `errcheck`, `govet`, `staticcheck`,
  `unused`, `ineffassign`, `gosimple`. Consider `revive`, `gocritic`, and
  `exhaustive` for stricter checks.
- `go test -race ./...` — race detector.

**Do not flag in review:**
- Formatting or import ordering (handled by `gofmt`/`goimports`).
- Unused variables or imports (handled by the compiler and linters).
- Simple linter-detectable issues (unreachable code, unnecessary conversions).
- Minor stylistic preferences that are already consistent within the codebase
  and not covered by the idiom rules above.

---

## 11 · Won't Fix — Explicitly Out of Scope

The following must **never** be flagged in a review. Raising these creates noise,
wastes time, and prevents reviews from converging.

- **Alternative approaches that are not clearly better.** "You could also do
  this with channels instead of a mutex" is not a finding unless the current
  approach has a concrete problem.
- **Variable renames** unless the current name is actively misleading or collides
  with a builtin.
- **"Consider adding a comment"** on code that is already clear from context.
- **Speculative performance optimisation** without evidence of a bottleneck
  (profiling data, benchmark results, or a known hot path).
- **Suggesting dependencies** the project doesn't already use. Don't recommend
  adding libraries, frameworks, or tools unless there is a clear, present deficiency
  that the standard library and existing dependencies cannot address.
- **Authentication, authorisation, and rate limiting.** These are intentionally
  excluded from this application's scope.
- **Bikeshedding** on file/package organisation that doesn't affect coupling or
  testability.
- **Rewriting working code** to a different but equivalent style. If it works,
  reads clearly, and passes linters, it's done.

---

## Convergence Rule

A codebase that satisfies all MUST FIX and SHOULD FIX criteria in this document,
passes its linter suite, and has reasonable test coverage for its public API
surface is **review-complete**. Subsequent review passes that surface only
CONSIDER-level or out-of-scope items are a signal to stop reviewing and ship.
