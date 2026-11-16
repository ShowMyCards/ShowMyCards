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

| Package                                                          | Pinned  | Advisory                                                                                                                       | Severity | Parent (why we need it)                                                                                                                                                                                                                                            | Remove when                                                            |
| ---------------------------------------------------------------- | ------- | ------------------------------------------------------------------------------------------------------------------------------ | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ---------------------------------------------------------------------- |
| `postcss-selector-parser` (under `@tailwindcss/typography` only) | `6.1.6` | [GHSA-rj75-hqrm-r3gf](https://github.com/advisories/GHSA-rj75-hqrm-r3gf) — quadratic complexity in flat selector parsing (DoS) | moderate | `@tailwindcss/typography` pins exactly `6.0.10`. Scoped to typography so `eslint-plugin-svelte › svelte-eslint-parser` keeps its own `postcss-selector-parser 7.x`; a global override would force it down a major. Scoped overrides need the v3 `bun.lock` format. | `@tailwindcss/typography` depends on `postcss-selector-parser >=6.1.5` |

## Removed overrides

Removed with the SvelteKit 3 / Vite 8 upgrade, after checking each with the procedure below:

- `cookie` — Kit 3 depends on `cookie` v2, so the old `^0.7.2` pin would have forced it back to an incompatible major.
- `esbuild` — nothing in the tree depends on it any more; Vite 8 bundles with rolldown.
- `vite` — Kit 3 requires Vite 8, which is a direct devDependency and resolves to a single patched version.

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
