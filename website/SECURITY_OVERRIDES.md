# Website Security Overrides

The `overrides` block in `package.json` force-resolves transitive dependencies
to patched versions where the parent dependency still pins a vulnerable range.

This file is the source of truth for **why** each override exists and **when**
it can be removed.

## Active overrides

None. `package.json` currently has no `overrides` block.

## Removed overrides

Removed with the Astro 7 upgrade, after checking each with the procedure below:

- `esbuild` — only `wrangler` uses it now, and it pins the fixed `0.28.2`.
- `js-yaml` — no longer in the dependency tree.
- `vite` — Astro 7 requires Vite 8, and the old `^7.3.6` override would have blocked it.

Known dev-only finding: `@tailwindcss/typography` pins a vulnerable
`postcss-selector-parser` (GHSA-rj75-hqrm-r3gf). CI audits `--prod` only, so it
isn't overridden here. The frontend's scoped override would move `bun.lock` to
the v3 format, which the Cloudflare Workers Builds image may not be able to read.

When adding an override, document it in a table with these columns:

| Package | Pinned | Advisory | Severity | Parent (why we need it) | Remove when |
| --- | --- | --- | --- | --- | --- |

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
