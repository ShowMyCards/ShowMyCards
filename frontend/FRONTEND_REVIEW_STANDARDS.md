# Code Review Standards — Svelte Frontend

> **Purpose:** Define a finite, repeatable rubric for AI-assisted code reviews
> of the Svelte 5 / SvelteKit frontend. Reviews check _only_ what is listed
> here. If an issue does not fall into a category below, it is out of scope and
> must not be flagged.

---

## How to Use This Document

When performing a code review, work through each section in order. For every
finding, assign exactly one severity from the table below. Do not invent new
severities or categories.

| Severity | Meaning | Action |
|----------|---------|--------|
| **MUST FIX** | Bugs, broken reactivity, accessibility failures, data leaks to client | Block merge |
| **SHOULD FIX** | Meaningful improvements to maintainability, idiom conformance, or component design | Fix in this PR or create a tracked issue |
| **CONSIDER** | Minor suggestions that are genuinely better but where the current code is acceptable | Author's discretion — no follow-up required |

Present findings as a structured list grouped by severity. Each finding must
reference a specific file and line range, state what the problem is, explain
_why_ it matters, and suggest a concrete fix.

---

## 1 · Correctness & Runtime Safety

These are bugs or near-bugs. Always flag.

- **Broken reactivity.** State that should update the DOM but doesn't, typically
  caused by mutating values without `$state`, reading stale closures, or using
  `$derived` where `$derived.by` is needed for complex expressions.
- **Unhandled promise rejections.** Async operations in `load` functions,
  `$effect`, or event handlers that lack error handling. Uncaught errors in
  `load` will surface as SvelteKit error pages; uncaught errors in client-side
  code will silently fail.
- **Memory leaks.** `$effect` blocks that set up subscriptions, intervals, or
  event listeners without returning a cleanup function. Any manual
  `addEventListener` must have a corresponding removal path.
- **Server/client boundary violations.** Importing server-only modules
  (`$env/static/private`, database clients, secrets) into client-side code or
  shared components. SvelteKit will error on this, but barrel exports and
  re-exports can mask it.
- **Unsafe HTML rendering.** Using `{@html ...}` with user-provided or
  API-sourced content without sanitisation is an XSS vector regardless of
  whether the app is local-only.

---

## 2 · Svelte 5 Idioms & Runes

The codebase must use Svelte 5 runes exclusively. Flag any Svelte 4 patterns.

### Must Use (Svelte 5)

| Pattern | Rune |
|---------|------|
| Reactive state | `let x = $state(initial)` |
| Derived values | `let y = $derived(expr)` or `$derived.by(() => ...)` |
| Side effects | `$effect(() => { ... })` |
| Component props | `let { prop1, prop2 } = $props()` |
| Bindable props | `let { value = $bindable() } = $props()` |
| Event callbacks | Pass as props: `let { onclick } = $props()` |
| Slots / children | `{@render children()}` and `{@render namedSlot?.()}` |

### Must NOT Use (Svelte 4 — flag as MUST FIX)

- `export let` for props
- `$:` reactive declarations or reactive statements
- `createEventDispatcher()` or `on:event` directive syntax
- `<slot />` or `<slot name="x" />`
- `$$props`, `$$restProps`, `$$slots`
- Svelte stores (`writable`, `readable`, `derived` from `svelte/store`) for
  local or shared component state — use `$state` in `.svelte.js`/`.svelte.ts`
  files instead

### Rune Usage Guidelines (SHOULD FIX if violated)

- **Prefer `$derived` over `$effect` for computed values.** If something can be
  expressed as a pure derivation, it must not be an effect that writes to
  another `$state` variable. `$effect` is an escape hatch, not a general
  reactivity tool.
- **Keep `$effect` side-effect-only.** Effects should interact with external
  systems (DOM APIs, `fetch`, logging, `localStorage`). They should not be used
  to synchronise two pieces of `$state` — use `$derived` instead.
- **Return cleanup from `$effect` when needed.** If the effect sets up a
  subscription, interval, or listener, return a cleanup function.
- **Don't nest `$state` unnecessarily.** For simple values (strings, numbers,
  booleans), plain `$state(value)` is sufficient. For objects and arrays,
  `$state` provides deep reactivity automatically.

---

## 3 · Component Design

- **Single responsibility.** A component should represent one coherent piece of
  UI or behaviour. If a component file exceeds ~150 lines, consider whether it
  is doing too much.
- **Props as the API surface.** Components communicate downward via props and
  upward via callback props. Avoid reaching into parent/child internals.
- **Extract shared logic into `.svelte.js` / `.svelte.ts` files.** Reusable
  reactive logic (state machines, data fetching patterns, form validation)
  should live in rune-enabled modules, not duplicated across components.
- **Avoid prop drilling beyond 2–3 levels.** For state that many distant
  components need, use Svelte's `setContext` / `getContext` or a shared
  `.svelte.js` module with `$state`.
- **Keep `{#each}` blocks keyed.** When rendering lists of items that can
  change order or be filtered, always use `{#each items as item (item.id)}`.
  Missing keys cause subtle reuse bugs.

---

## 4 · Data Fetching & API Integration

Since the frontend consumes a Go Fiber API:

- **Centralise API calls.** All fetch calls to the backend should go through a
  shared API client module (`$lib/api/` or similar). Do not scatter raw `fetch`
  calls across components.
- **Type the API responses.** Define TypeScript interfaces/types for every API
  response shape. Do not use `any` or untyped `data` objects.
- **Handle loading, error, and empty states.** Every data-fetching component or
  `load` function must account for all three states. Don't render content with
  `undefined` data.
- **Use SvelteKit `load` functions for page data.** Data needed at page render
  should come from `+page.ts` / `+page.server.ts` load functions, not from
  `$effect` in the component. Client-side `$effect` fetching is appropriate for
  subsequent user-driven requests (search, pagination, etc.) but not initial
  page data.
- **Don't hardcode the API base URL.** The backend URL should be configurable
  (via `$env/static/public` or a build-time variable), not hardcoded as
  `http://localhost:3000`.

---

## 5 · TypeScript Usage

- **Enable strict mode.** `tsconfig.json` should have `"strict": true`. If it
  doesn't, flag as SHOULD FIX.
- **No `any`.** Every use of `any` is SHOULD FIX unless there is a documented
  reason (e.g. a third-party library with no types). Prefer `unknown` when the
  type is genuinely uncertain, then narrow.
- **Type component props explicitly.** Use TypeScript types in `$props()`:
  `let { name, count }: { name: string; count: number } = $props()` or define
  a separate `Props` type.
- **Type `load` function return values.** SvelteKit infers types from `load`
  functions via `PageData` / `LayoutData`, but the load function itself should
  return well-typed objects — not `Record<string, unknown>` or untyped literals.
- **Prefer `satisfies` over `as` for type assertions.** `as` silences the type
  checker; `satisfies` validates while preserving the narrower type.

---

## 6 · Structure & Decomposition

### When to Flag

- **Components exceeding ~150 lines.** Long components often mix concerns
  (data fetching, state management, presentation). Split into container/
  presentational components or extract logic into `.svelte.js` modules.
- **Duplicated markup or logic appearing 3+ times.** Extract into a shared
  component or utility module.
- **Mixed abstraction levels in a single component.** A component that fetches
  data, transforms it, manages form state, and renders complex UI should be
  decomposed. Separate data logic from presentation.
- **Overly complex template expressions.** If a `{#if}` / `{#each}` / `{:else}`
  block is nested 3+ levels deep, extract inner blocks into child components.
- **Utility grab-bags.** A `$lib/utils.ts` file with 20+ unrelated exports
  should be broken into focused modules.

### When NOT to Flag

- **Extracting a component used exactly once** that doesn't clarify the code.
- **Moving files between directories** without changing the component API or
  reducing coupling.
- **Suggesting additional abstractions** (wrapper components, higher-order
  patterns) for code that has one concrete use case.

---

## 7 · Styling

- **Scoped styles by default.** Use Svelte's built-in `<style>` block, which
  scopes CSS to the component. Avoid `:global()` unless explicitly needed for
  third-party component overrides.
- **No inline styles for static values.** Prefer CSS classes over `style=""`
  attributes when the styles don't change dynamically.
- **Use CSS custom properties for theming.** Prefer `--var` custom properties
  over hardcoded colours/spacing values scattered across components.
- **Responsive design.** Flag fixed pixel widths on containers that should be
  fluid. Flag missing mobile considerations only if the breakpoint is clearly
  broken.

---

## 8 · Accessibility

Accessibility issues are MUST FIX. A self-hosted local app still needs to be
usable.

- **Semantic HTML.** Use `<button>` for actions, `<a>` for navigation, `<nav>`,
  `<main>`, `<header>`, `<section>` where appropriate. Don't use `<div>` with
  `onclick` as a button substitute.
- **Form labels.** Every `<input>`, `<select>`, and `<textarea>` must have an
  associated `<label>` (via `for`/`id` or wrapping). Placeholder text is not
  a label.
- **Alt text.** Every `<img>` needs an `alt` attribute. Decorative images use
  `alt=""`.
- **Keyboard navigation.** Interactive elements must be reachable and operable
  via keyboard. Custom interactive elements need appropriate `tabindex`, `role`,
  and ARIA attributes.
- **Colour contrast.** Text must meet WCAG AA contrast ratios against its
  background. Flag obviously low-contrast text combinations.

---

## 9 · Error Handling & Edge Cases

- **`load` function errors.** Use SvelteKit's `error()` helper to throw typed
  errors. Don't let raw exceptions propagate — they produce unhelpful error
  pages.
- **`+error.svelte` pages.** The app should have at minimum a root-level error
  page that handles unexpected failures gracefully.
- **Form validation.** Client-side validation should exist for all user inputs.
  Don't rely solely on the backend for validation feedback.
- **Empty states.** When a list or dataset is empty, show an intentional empty
  state — not a blank page or a broken layout.
- **Loading indicators.** Long-running operations should show progress. Don't
  leave the user staring at an unchanged screen during fetches.

---

## 10 · SvelteKit Conventions

- **File-based routing.** Routes live in `src/routes/`. Don't create a custom
  routing system alongside SvelteKit's.
- **`+page.ts` vs `+page.server.ts`.** Use `+page.server.ts` when the load
  function accesses secrets or server-only resources. Use `+page.ts` when the
  load function can run in both environments. Don't default to server-side when
  client-side is sufficient.
- **Layout hierarchy.** Use `+layout.svelte` and `+layout.ts` for shared UI
  and data. Don't repeat the same wrapper markup in every page component.
- **`$lib` alias.** All shared code should live under `src/lib/` and be
  imported via `$lib/...`. Don't use relative paths like `../../../lib/`.
- **Environment variables.** Use `$env/static/public` for client-safe config
  and `$env/static/private` for server-only secrets. Don't use `process.env`
  or `import.meta.env` directly.

---

## 11 · Linter & Tooling Baseline

These items are handled by automated tooling and **must not be flagged in
review**. Ensure the following are configured:

- `eslint` with `eslint-plugin-svelte` — linting for Svelte and JS/TS.
- `svelte-check` — type checking and Svelte-specific diagnostics.
- `prettier` with `prettier-plugin-svelte` — formatting.
- `typescript` in strict mode — type safety.

**Do not flag in review:**
- Formatting, indentation, or import ordering (handled by Prettier).
- Unused imports or variables (handled by ESLint / `svelte-check`).
- TypeScript errors that would be caught by `svelte-check`.
- Minor stylistic preferences already consistent within the codebase.

---

## 12 · Won't Fix — Explicitly Out of Scope

The following must **never** be flagged in a review. Raising these creates noise
and prevents reviews from converging.

- **Alternative component libraries or frameworks.** Don't suggest switching
  from the current UI approach.
- **SSR vs CSR strategy debates** unless there's a concrete performance or SEO
  issue (and since this is a local app consuming a local API, SSR is largely
  irrelevant).
- **Micro-optimisations** without evidence of a performance problem (e.g. "use
  `{@const}` to avoid re-evaluation" when the expression is trivial).
- **Suggesting new dependencies** the project doesn't already use.
- **Bikeshedding on naming** unless a name is actively misleading.
- **Authentication, authorisation, and rate limiting.** Excluded from scope.
- **Rewriting working components** to a different but equivalent pattern. If it
  works, reads clearly, passes linters, and uses Svelte 5 runes, it's done.
- **Speculative "what if you need X later"** abstractions.

---

## Convergence Rule

A codebase that satisfies all MUST FIX and SHOULD FIX criteria in this document,
passes its linter suite (`eslint`, `svelte-check`, `tsc`), and has reasonable
test coverage for critical user flows is **review-complete**. Subsequent review
passes that surface only CONSIDER-level or out-of-scope items are a signal to
stop reviewing and ship.
