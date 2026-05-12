<!--
Thanks for opening a pull request!

The PR title should follow Conventional Commits — it becomes the squash
commit subject and drives automated versioning:
  feat: <user-visible change>
  fix:  <user-visible bug fix>
  chore|docs|refactor|test|ci: <internal change, no release>
Append `!` to indicate a breaking change: `feat!: ...`
-->

## What does this change?

<!-- One paragraph. What does the diff actually do, and why? -->

## Why are we doing this?

<!-- What problem does this solve, or what use case does it enable?
     Link the issue this PR closes, if any: `Closes #123` -->

## How was this tested?

<!-- The CI runs tests, lint, type checks, and vulnerability scans
     automatically. Describe any additional manual testing you did, especially
     for UI changes — what flows did you click through? -->

## Checklist

- [ ] PR title follows Conventional Commits (`feat:`, `fix:`, etc.)
- [ ] Changes are focused on one logical thing
- [ ] If backend models or API response types changed, `make types` was run
- [ ] No new auth, rate limiting, or SaaS dependencies introduced
- [ ] I have read [CONTRIBUTING.md](../CONTRIBUTING.md)
