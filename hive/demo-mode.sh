#!/usr/bin/env bash
# Puts Hive in demo mode, after you have signed in to Claude once from the dashboard.
#
# Hive is built to run continuously, not to be tried in an afternoon. This does the things the
# walkthrough should not have to:
#
#   - Skips Claude Code's first-run screens for every agent. Each agent is its own Claude Code
#     instance with its own settings (which is what lets each use a different model), and each
#     would stop once on "Security notes … Press Enter to continue" and on a Bypass Permissions
#     consent whose default answer is "No, exit". Hive does not answer these for Claude agents.
#   - Lets scanner comment on the issues it diagnoses, which Hive normally allows only from L4.
#   - Restarts every agent, so it picks up the sign-in now rather than in a few minutes.
#   - Makes the governor check every minute instead of every ten, so findings reach the advisory
#     issue soon after an agent finishes. This takes effect after the governor's current wait.
#
# Run it again after promoting to L3, for the agent that level adds.
#
#   hive/demo-mode.sh
#
# Needs HIVE_DASHBOARD_TOKEN, and Hive running as the hive-demo container.
set -euo pipefail

: "${HIVE_DASHBOARD_TOKEN:?set HIVE_DASHBOARD_TOKEN}"

api=http://127.0.0.1:3001/api
auth="Authorization: Bearer $HIVE_DASHBOARD_TOKEN"

# Each file is edited as its owner: root in the container cannot write them.
#
# settings.json is shared by every agent. skipDangerousModePermissionPrompt stops Claude Code asking
# each agent to accept Bypass Permissions mode.
# .claude.json is per agent. hasCompletedOnboarding skips the first-run screens.
agents="$(docker exec hive-demo bash -c '
s=/data/home/.claude/settings.json
su -s /bin/bash "$(stat -c %U "$s")" -c "jq \".skipDangerousModePermissionPrompt = true\" \"$s\" > \"$s.tmp\" && cat \"$s.tmp\" > \"$s\" && rm \"$s.tmp\""
for f in /data/home/agents/*/.claude.json; do
  su -s /bin/bash "$(stat -c %U "$f")" -c "umask 077; jq \".hasCompletedOnboarding = true\" \"$f\" > \"$f.tmp\" && mv \"$f.tmp\" \"$f\""
  basename "$(dirname "$f")"
done
')"

# Lets scanner comment on the issues it diagnoses. Hive's L2 keeps scanner advisory (its findings
# only reach the advisory report) and allows issue comments from L4. In the ACMM, L2 means acting on
# issues and PRs, as reviewer already does on PRs. Set here because the L2 pack resets it on first
# start; the restart below applies it.
curl -fsS -X PUT -H "$auth" -H "Content-Type: application/json" \
  -d '{"mode":"ISSUES_ONLY","kickTemplate":"scanner-issues.md"}' "$api/config/agent/scanner/general" >/dev/null
echo "scanner: comments on issues"

for agent in $agents; do
  printf '%s: ' "$agent"
  curl -fsS -X POST -H "$auth" "$api/restart/$agent" >/dev/null && echo "ready"
done

curl -fsS -X PUT -H "$auth" -H "Content-Type: application/json" \
  -d '{"eval_interval_s":60}' "$api/config/governor/general-advanced" >/dev/null
echo "governor: checking every minute"
