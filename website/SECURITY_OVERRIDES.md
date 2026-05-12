# Website Security Overrides

The `overrides` block in `package.json` force-resolves transitive dependencies
to patched versions where the parent dependency still pins a vulnerable range.
Each entry exists because `bun audit` flagged a finding that could not be
resolved by `bun update` alone.

This file is the source of truth for **why** each override exists and **when**
it can be removed. Renovate will propose version bumps for these entries as new
releases come out — when reviewing those PRs, check this file to understand
whether the bump is required or just a follow-along.

## Active overrides

| Package     | Pinned    | Advisory                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            | Severity | Parent (why we need it)                                                                       | Remove when                                                              |
| ----------- | --------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- | --------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------ |
| `picomatch` | `^4.0.4`  | [GHSA-c2c7-rcm5-vvqj](https://github.com/advisories/GHSA-c2c7-rcm5-vvqj) + [GHSA-3v7f-55p6-f55p](https://github.com/advisories/GHSA-3v7f-55p6-f55p) — ReDoS + method injection                                                                                                                                                                                                                                                                                                                                                                                      | high     | `astro › @rollup/pluginutils`, `@astrojs/cloudflare › vite › tinyglobby › fdir`, `@tailwindcss/vite › vite › tinyglobby › fdir` | upstream `@rollup/pluginutils` / `tinyglobby` bump `picomatch` to `>=4.0.4` |
| `postcss`   | `^8.5.10` | [GHSA-qx2v-qp2m-jg93](https://github.com/advisories/GHSA-qx2v-qp2m-jg93) — XSS via unescaped `</style>`                                                                                                                                                                                                                                                                                                                                                                                                                                                              | moderate | `astro › vite`, `@astrojs/cloudflare › vite`, `@tailwindcss/vite › vite`                       | vite consumers bump `postcss` to `>=8.5.10`                              |
| `rollup`    | `^4.59.0` | [GHSA-mw96-cpmx-2vgc](https://github.com/advisories/GHSA-mw96-cpmx-2vgc) — arbitrary file write via path traversal                                                                                                                                                                                                                                                                                                                                                                                                                                                  | high     | `vite › rollup` (all three vite chains)                                                       | `vite` bumps `rollup` to `>=4.59.0`                                       |
| `svgo`      | `^4.0.1`  | [GHSA-xpqw-6gx7-v673](https://github.com/advisories/GHSA-xpqw-6gx7-v673) — DoS via DOCTYPE entity expansion (Billion Laughs)                                                                                                                                                                                                                                                                                                                                                                                                                                          | high     | `astro › svgo`                                                                                | `astro` bumps `svgo` to `>=4.0.1`                                         |
| `undici`    | `^7.18.2` | [GHSA-g9mf-h72j-4rw9](https://github.com/advisories/GHSA-g9mf-h72j-4rw9), [GHSA-f269-vfmq-vjvj](https://github.com/advisories/GHSA-f269-vfmq-vjvj), [GHSA-2mjp-6q6p-2qxm](https://github.com/advisories/GHSA-2mjp-6q6p-2qxm), [GHSA-vrm6-8vpv-qv8q](https://github.com/advisories/GHSA-vrm6-8vpv-qv8q), [GHSA-v9p9-hfj2-hcw8](https://github.com/advisories/GHSA-v9p9-hfj2-hcw8), [GHSA-4992-7rv2-5pvq](https://github.com/advisories/GHSA-4992-7rv2-5pvq) — WebSocket overflow, decompression exhaustion, request smuggling, CRLF injection, etc. | high     | `wrangler › miniflare › undici`                                                               | `wrangler`/`miniflare` bumps `undici` to `>=7.18.2`                       |

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

Renovate sees `overrides` entries as normal deps and will propose bumps (e.g.
`undici 7.x → 8.x`, `picomatch 4.x → 5.x`). Before approving:

1. Check whether the consumers in the **Parent** column also moved to that
   major. If they did, the override is probably unnecessary now — remove it
   instead of bumping it.
2. If the consumers still pin the older major, the override bump only helps if
   the new major's API is compatible with what the consumer uses. Verify by
   running `bun run build` and the deploy dry-run (`bunx wrangler deploy
   --dry-run`) with the bump.
