# Frontend Security Overrides

The `overrides` block in `package.json` force-resolves transitive dependencies
to patched versions where the parent dependency still pins a vulnerable range.
Each entry exists because `bun audit` flagged a finding that could not be
resolved by `bun update` alone.

This file is the source of truth for **why** each override exists and **when**
it can be removed. Renovate will propose version bumps for these entries as new
releases come out — when reviewing those PRs, check this file to understand
whether the bump is required or just a follow-along.

## Active overrides

| Package   | Pinned    | Advisory                                                                                                                                               | Severity | Parent (why we need it)                       | Remove when                                                                                                         |
| --------- | --------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ | -------- | --------------------------------------------- | ------------------------------------------------------------------------------------------------------------------- |
| `cookie`  | `^0.7.2`  | [GHSA-pxg6-pf52-xh8x](https://github.com/advisories/GHSA-pxg6-pf52-xh8x) — out-of-bounds chars accepted                                                | low      | `@sveltejs/kit` (pins `cookie ^0.6.0`)        | `@sveltejs/kit` bumps its `cookie` pin to `^0.7.0` or later                                                         |
| `esbuild` | `^0.28.1` | [GHSA-gv7w-rqvm-qjhr](https://github.com/advisories/GHSA-gv7w-rqvm-qjhr) — missing binary integrity verification enables RCE via `NPM_CONFIG_REGISTRY` | high     | `vite › esbuild` (resolves `esbuild <0.28.1`) | `vite`/`vitest` stop resolving an `esbuild <0.28.1` (no 0.27.x backport exists; `0.28.1` is the only fixed release) |

## How to check if an override is still required

Remove the entry from `package.json`, then:

```bash
bun install
bun audit --prod
```

- If `No vulnerabilities found` — the override is no longer needed; delete it.
- If the advisory reappears — revert; the parent dep hasn't caught up yet.

Do this for one override at a time so you know which one was load-bearing.

## When Renovate proposes a major bump for an entry here

Renovate sees `overrides` entries as normal deps and will propose bumps. Before
approving a major (e.g. `yaml 1.x → 2.x`, `cookie 0.x → 1.x`, `picomatch 4.x →
5.x`):

1. Check whether the consumers in the **Parent** column also moved to that
   major. If they did, the override is probably unnecessary now — remove it
   instead of bumping it.
2. If the consumers still pin the older major, the override bump only helps if
   the new major's API is compatible with what the consumer uses. Verify by
   running the build + tests with the bump.
