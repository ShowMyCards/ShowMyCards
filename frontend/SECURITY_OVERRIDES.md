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

| Package           | Pinned    | Advisory                                                                                                                                                                       | Severity | Parent (why we need it)                                                                                                                    | Remove when                                                                                    |
| ----------------- | --------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------- | ------------------------------------------------------------------------------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------- |
| `brace-expansion` | `^5.0.5`  | [GHSA-f886-m6hf-6m8v](https://github.com/advisories/GHSA-f886-m6hf-6m8v) — zero-step sequence DoS                                                                              | moderate | `eslint › @eslint/config-array › minimatch`, `typescript-eslint › … › minimatch`                                                           | both consumers ship a `minimatch` that uses `brace-expansion >=5.0.5`                          |
| `cookie`          | `^0.7.2`  | [GHSA-pxg6-pf52-xh8x](https://github.com/advisories/GHSA-pxg6-pf52-xh8x) — out-of-bounds chars accepted                                                                        | low      | `@sveltejs/kit` (pins `cookie ^0.6.0`)                                                                                                     | `@sveltejs/kit` bumps its `cookie` pin to `^0.7.0` or later                                    |
| `esbuild`         | `^0.28.1` | [GHSA-gv7w-rqvm-qjhr](https://github.com/advisories/GHSA-gv7w-rqvm-qjhr) — missing binary integrity verification enables RCE via `NPM_CONFIG_REGISTRY`                         | high     | `vite › esbuild` (resolves `esbuild <0.28.1`)                                                                                              | `vite`/`vitest` stop resolving an `esbuild <0.28.1` (no 0.27.x backport exists; `0.28.1` is the only fixed release) |
| `flatted`         | `^3.4.0`  | [GHSA-25h7-pfq9-p65f](https://github.com/advisories/GHSA-25h7-pfq9-p65f) + [GHSA-rf6f-7fwh-wjgh](https://github.com/advisories/GHSA-rf6f-7fwh-wjgh) — DoS + proto pollution    | high     | `eslint › file-entry-cache › flat-cache`                                                                                                   | `eslint` (or `flat-cache`) bumps `flatted` to `>=3.4.0`                                        |
| `picomatch`       | `^4.0.4`  | [GHSA-c2c7-rcm5-vvqj](https://github.com/advisories/GHSA-c2c7-rcm5-vvqj) + [GHSA-3v7f-55p6-f55p](https://github.com/advisories/GHSA-3v7f-55p6-f55p) — ReDoS + method injection | high     | `vite`, `vitest`, `svelte-check`, `typescript-eslint`, `@sveltejs/adapter-node` (all via `tinyglobby › fdir` or `@rollup/plugin-commonjs`) | upstream `tinyglobby`/`fdir`/`@rollup/plugin-commonjs` consumers bump `picomatch` to `>=4.0.4` |
| `postcss`         | `^8.5.10` | [GHSA-qx2v-qp2m-jg93](https://github.com/advisories/GHSA-qx2v-qp2m-jg93) — XSS via unescaped `</style>`                                                                        | moderate | `vite`, `eslint-plugin-svelte`                                                                                                             | both consumers bump `postcss` to `>=8.5.10`                                                    |
| `yaml`            | `^1.10.3` | [GHSA-48c2-rrv3-qjmp](https://github.com/advisories/GHSA-48c2-rrv3-qjmp) — stack overflow on deep nesting                                                                      | moderate | `vite › yaml`, `eslint-plugin-svelte › postcss-load-config › yaml` (yaml@1.x chain)                                                        | the chain drops `yaml@1` or bumps to `>=1.10.3` (or `>=2.x` — note major API change)           |

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
