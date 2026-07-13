#!/usr/bin/env bash
# One-shot script to apply the repository settings and branch protection that
# go alongside the release pipeline overhaul.
#
# Run this AFTER the chore/release-pipeline PR has been merged to main AND the
# new workflows (CI, Security, CodeQL) have run at least once — branch
# protection cannot require status check names that GitHub has never seen.
#
# Also install the Renovate GitHub App on the org before running this:
# https://github.com/apps/renovate. Step 4 names the app as a bypass actor, and
# GitHub will not accept a bypass actor for an app that is not installed.
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

# `can_approve_pull_request_reviews` is a misleadingly-named field: it
# controls both PR *creation* and PR approval by Actions, not just approval.
# It MUST be true here so release-please can open the "chore: release" PR
# that drives our entire release flow. Branch protection's
# `require_code_owner_reviews: true` is what actually prevents Actions from
# self-approving, since the bot is not in CODEOWNERS.
gh api -X PUT "repos/$REPO/actions/permissions/workflow" \
  -F default_workflow_permissions=read \
  -F can_approve_pull_request_reviews=true > /dev/null

echo "==> 4. Branch protection on main (rulesets)"
#
# We use two rulesets rather than one, and rather than legacy branch protection.
#
# The reason is a hard constraint in GitHub's model: a ruleset's bypass_actors
# bypass the WHOLE ruleset. There is no per-rule bypass. Renovate has to skip
# the human-review requirement (it cannot approve its own PRs, and CODEOWNERS
# maps * to a human), but it must NOT skip CI — govulncheck and bun audit are
# the checks that stop a vulnerable dependency reaching main, and a Renovate PR
# is exactly the PR we most want them to run on.
#
# Splitting them is the only way to express "Renovate skips review, Renovate
# still obeys CI":
#
#   main-guardrails  — status checks, linear history, no force-push/delete.
#                      NO bypass actors. Binds everyone, Renovate included.
#   main-review      — 1 approving CODEOWNER review.
#                      Renovate (app 2740) bypasses this one only.
#
RENOVATE_APP_ID=2740

# Idempotent create-or-update: rulesets have no PUT-by-name, so look up the id.
apply_ruleset() {
  local name="$1" payload="$2" id
  id=$(gh api "repos/$REPO/rulesets" --jq ".[] | select(.name == \"$name\") | .id")
  if [[ -n "$id" ]]; then
    gh api -X PUT "repos/$REPO/rulesets/$id" --input - <<<"$payload" > /dev/null
    echo "    updated ruleset '$name' ($id)"
  else
    gh api -X POST "repos/$REPO/rulesets" --input - <<<"$payload" > /dev/null
    echo "    created ruleset '$name'"
  fi
}

# Required status checks must match the job names GitHub has actually seen.
# integration_id 15368 is the GitHub Actions app — it owns all of these.
apply_ruleset "main-guardrails" "$(cat <<'JSON'
{
  "name": "main-guardrails",
  "target": "branch",
  "enforcement": "active",
  "conditions": { "ref_name": { "include": ["~DEFAULT_BRANCH"], "exclude": [] } },
  "bypass_actors": [],
  "rules": [
    { "type": "deletion" },
    { "type": "non_fast_forward" },
    { "type": "required_linear_history" },
    {
      "type": "required_status_checks",
      "parameters": {
        "strict_required_status_checks_policy": true,
        "required_status_checks": [
          { "context": "backend",                        "integration_id": 15368 },
          { "context": "frontend",                       "integration_id": 15368 },
          { "context": "govulncheck (backend)",          "integration_id": 15368 },
          { "context": "bun audit (frontend)",           "integration_id": 15368 },
          { "context": "bun audit (website)",            "integration_id": 15368 },
          { "context": "actionlint (workflows)",         "integration_id": 15368 },
          { "context": "zizmor (workflow security)",     "integration_id": 15368 },
          { "context": "Analyze (go)",                   "integration_id": 15368 },
          { "context": "Analyze (javascript-typescript)","integration_id": 15368 }
        ]
      }
    }
  ]
}
JSON
)"

apply_ruleset "main-review" "$(cat <<JSON
{
  "name": "main-review",
  "target": "branch",
  "enforcement": "active",
  "conditions": { "ref_name": { "include": ["~DEFAULT_BRANCH"], "exclude": [] } },
  "bypass_actors": [
    { "actor_id": $RENOVATE_APP_ID, "actor_type": "Integration", "bypass_mode": "always" }
  ],
  "rules": [
    {
      "type": "pull_request",
      "parameters": {
        "required_approving_review_count": 1,
        "require_code_owner_review": true,
        "dismiss_stale_reviews_on_push": true,
        "require_last_push_approval": false,
        "required_review_thread_resolution": true,
        "allowed_merge_methods": ["squash"]
      }
    }
  ]
}
JSON
)"

echo "==> 5. Remove legacy branch protection on main"
# This MUST happen, and must happen last. Legacy branch protection and rulesets
# are additive — the most restrictive wins — and legacy protection has no
# concept of an app bypass. Leaving it in place would keep enforcing the
# 1-review requirement against Renovate and silently undo the whole exercise.
# 404 is the success case on a re-run, so don't let it trip set -e.
gh api -X DELETE "repos/$REPO/branches/main/protection" > /dev/null 2>&1 \
  && echo "    legacy branch protection deleted" \
  || echo "    no legacy branch protection present (already migrated)"

echo
echo "Done. Verify in the GitHub UI:"
echo "  https://github.com/$REPO/settings/rules"
echo "  https://github.com/$REPO/settings/security_analysis"
echo "  https://github.com/$REPO/settings/actions"
echo
echo "If a required status check is missing from the list, the workflow has"
echo "not yet run on main — push a no-op commit or wait for the next merge,"
echo "then re-run this script."
echo
echo "The 'main-review' bypass actor only resolves if the Renovate GitHub App"
echo "is installed on the org. Install it first: https://github.com/apps/renovate"
