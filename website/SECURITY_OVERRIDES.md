# Website Security Overrides

The `overrides` block in `package.json` force-resolves transitive dependencies
to patched versions where the parent dependency still pins a vulnerable range.

This file is the source of truth for **why** each override exists and **when**
it can be removed.

## Active overrides

| Package | Pinned | Advisory | Severity | Parent (why we need it) | Remove when |
| --- | --- | --- | --- | --- | --- |
| `devalue` | `^5.8.1` | [GHSA-77vg-94rm-hx3p](https://github.com/advisories/GHSA-77vg-94rm-hx3p) — DoS via sparse array deserialization | High | `astro` declares `devalue: "^5.6.3"`, but `bun update` leaves the transitive dep locked at `devalue@5.8.0`, inside the `>=5.6.3 <=5.8.0` vulnerable range. The override forces resolution to the patched `5.8.1`. | A fresh `bun install` with this entry removed resolves `devalue >= 5.8.1` — typically once Astro ships a release that locks devalue past the vulnerable range. Verify per "How to check if an override is still required" below. |
| `esbuild` | `^0.28.1` | [GHSA-gv7w-rqvm-qjhr](https://github.com/advisories/GHSA-gv7w-rqvm-qjhr) — missing binary integrity verification enables RCE via `NPM_CONFIG_REGISTRY` | High | `astro › esbuild` and `vite › esbuild` both resolve `esbuild <0.28.1` (0.27.7), inside the `>=0.17.0 <0.28.1` vulnerable range. The override forces resolution to the patched `0.28.1`. | `astro`/`vite` stop resolving an `esbuild <0.28.1` (no 0.27.x backport exists; `0.28.1` is the only fixed release). Verify per "How to check if an override is still required" below. |

## When to add an override

If `bun audit --prod` reports a finding in a transitive dep, first try:

```bash
bun update
bun audit --prod
```

If the finding remains because a parent dep pins a vulnerable range, add an
entry to `overrides` in `package.json` pinning the package to a patched
version, then document it in a table here with these columns:

| Package | Pinned | Advisory | Severity | Parent (why we need it) | Remove when |

The **Remove when** column should describe the upstream condition that makes
the override unnecessary — typically "parent dep bumps its pin to `>=fix`".

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

Renovate sees `overrides` entries as normal deps and will propose bumps.
Before approving a major bump:

1. Check whether the consumers in the **Parent** column also moved to that
   major. If they did, the override is probably unnecessary now — remove it
   instead of bumping it.
2. If the consumers still pin the older major, the override bump only helps if
   the new major's API is compatible with what the consumer uses. Verify by
   running `bun run build` with the bump.
