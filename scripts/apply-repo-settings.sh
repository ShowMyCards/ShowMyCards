#!/usr/bin/env bash
# One-shot script to apply the repository settings and branch protection that
# go alongside the release pipeline overhaul.
#
# Run this AFTER the chore/release-pipeline PR has been merged to main AND the
# new workflows (CI, Security, CodeQL) have run at least once — branch
# protection cannot require status check names that GitHub has never seen.
#
# Re-running the script is safe: every call is idempotent.

set -euo pipefail

REPO="ShowMyCards/ShowMyCards"

echo "==> 1. Repository merge + cleanup settings"
gh api -X PATCH "repos/$REPO" \
  -F allow_squash_merge=true \
  -F allow_merge_commit=false \
  -F allow_rebase_merge=false \
  -F delete_branch_on_merge=true \
  -F allow_auto_merge=true \
  -F allow_update_branch=true \
  -F squash_merge_commit_title=PR_TITLE \
  -F squash_merge_commit_message=PR_BODY \
  -F web_commit_signoff_required=false \
  > /dev/null

echo "==> 2. Secret scanning + push protection (free for public repos)"
gh api -X PATCH "repos/$REPO" \
  --input - <<'JSON' > /dev/null
{
  "security_and_analysis": {
    "secret_scanning": { "status": "enabled" },
    "secret_scanning_push_protection": { "status": "enabled" }
  }
}
JSON

echo "==> 3. Require approval for outside contributor workflow runs"
# This stops forks from auto-running workflows on first contact. Equivalent
# of the "Require approval for first-time contributors" radio button in the
# Actions settings UI.
gh api -X PUT "repos/$REPO/actions/permissions/access" \
  -F access_level=organization > /dev/null || true

gh api -X PUT "repos/$REPO/actions/permissions/workflow" \
  -F default_workflow_permissions=read \
  -F can_approve_pull_request_reviews=false > /dev/null

echo "==> 4. Branch protection on main"
# Required status checks must match the job names GitHub has actually seen.
# The names below come from the new workflow files in this PR.
gh api -X PUT "repos/$REPO/branches/main/protection" \
  --input - <<'JSON' > /dev/null
{
  "required_status_checks": {
    "strict": true,
    "contexts": [
      "backend",
      "frontend",
      "govulncheck (backend)",
      "bun audit (frontend)",
      "bun audit (website)",
      "actionlint (workflows)",
      "zizmor (workflow security)",
      "Analyze (go)",
      "Analyze (javascript-typescript)"
    ]
  },
  "enforce_admins": false,
  "required_pull_request_reviews": {
    "dismiss_stale_reviews": true,
    "require_code_owner_reviews": true,
    "required_approving_review_count": 1,
    "require_last_push_approval": false
  },
  "restrictions": null,
  "required_linear_history": true,
  "allow_force_pushes": false,
  "allow_deletions": false,
  "required_conversation_resolution": true,
  "lock_branch": false,
  "allow_fork_syncing": true
}
JSON

echo
echo "Done. Verify in the GitHub UI:"
echo "  https://github.com/$REPO/settings/branches"
echo "  https://github.com/$REPO/settings/security_analysis"
echo "  https://github.com/$REPO/settings/actions"
echo
echo "If a required status check is missing from the list, the workflow has"
echo "not yet run on main — push a no-op commit or wait for the next merge,"
echo "then re-run this script."
