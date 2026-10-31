#!/usr/bin/env bash
set -euo pipefail

# This probe deliberately creates two uniquely named lightweight tags, not Releases.
[[ "${GITHUB_REPOSITORY:-}" == jozala/omnigrex-testrun ]] || { echo 'Run only in jozala/omnigrex-testrun.' >&2; exit 1; }
[[ "${GITHUB_EVENT_NAME:-}" == workflow_dispatch && "${GITHUB_REF:-}" == refs/heads/main ]] || { echo 'Dispatch the workflow on main.' >&2; exit 1; }
[[ "${HISTORICAL_SHA:-}" =~ ^[0-9a-f]{40}$ ]] || { echo 'Supply a full historical commit SHA.' >&2; exit 1; }
: "${GITHUB_STEP_SUMMARY:?}"
: "${GITHUB_RUN_ID:?}"
: "${GITHUB_RUN_ATTEMPT:?}"
: "${GITHUB_SHA:?}"

[[ "$(git rev-parse HEAD)" == "$GITHUB_SHA" && "$(git rev-parse refs/remotes/origin/main)" == "$GITHUB_SHA" ]] || { echo 'The checkout must match the dispatched main tip.' >&2; exit 1; }
[[ "$HISTORICAL_SHA" != "$GITHUB_SHA" ]] || { echo 'The historical target must differ from the control.' >&2; exit 1; }
git merge-base --is-ancestor "$HISTORICAL_SHA" "$GITHUB_SHA" || { echo 'The historical target must be an ancestor of main.' >&2; exit 1; }
if git diff --quiet "$HISTORICAL_SHA" "$GITHUB_SHA" -- .github/workflows; then
  echo 'The historical target must have different workflow contents.' >&2
  exit 1
fi

# A successful previous probe makes the historical commit referenced and changes
# the experiment. Require manual cleanup before repeating that target.
while IFS='|' read -r direct peeled ref; do
  if [[ "$direct" == "$HISTORICAL_SHA" || "$peeled" == "$HISTORICAL_SHA" ]]; then
    echo "Historical target already has a direct branch/tag ref: $ref" >&2
    echo 'Remove only the previously created probe tag, or choose another unreferenced fixture.' >&2
    exit 1
  fi
done < <(git for-each-ref --format='%(objectname)|%(*objectname)|%(refname)' refs/remotes/origin refs/tags)

printf '# Historical tag permission probe\n\nRepository: `%s`\n\n' "$GITHUB_REPOSITORY" >> "$GITHUB_STEP_SUMMARY"
printf '| Target | Commit | Attempted tag | Result |\n| --- | --- | --- | --- |\n' >> "$GITHUB_STEP_SUMMARY"

probe() {
  local target="$1" commit="$2" status
  local tag="permission-probe-${GITHUB_RUN_ID}-${GITHUB_RUN_ATTEMPT}-${target}"
  printf '\nTesting %s: %s -> %s\n' "$target" "$tag" "$commit"
  # Print response headers/body for diagnosis, never the authentication token.
  # This is the same create-reference endpoint used by Omnigrex PR #161.
  if gh api --include --method POST "repos/${GITHUB_REPOSITORY}/git/refs" \
      -f "ref=refs/tags/${tag}" -f "sha=${commit}"; then
    printf '| %s | `%s` | `%s` | Created |\n' "$target" "$commit" "$tag" >> "$GITHUB_STEP_SUMMARY"
    return 0
  else
    status=$?
    printf '| %s | `%s` | `%s` | Request failed (gh exit %s; inspect HTTP response) |\n' "$target" "$commit" "$tag" "$status" >> "$GITHUB_STEP_SUMMARY"
    return "$status"
  fi
}

current=failed
historical=failed
if probe current "$GITHUB_SHA"; then current=created; fi
if probe historical "$HISTORICAL_SHA"; then historical=created; fi

printf '\nCreated tags are left for inspection. Before repeating this historical target, delete only the exact successful `permission-probe-*` tags listed above.\n\n' >> "$GITHUB_STEP_SUMMARY"
if [[ "$current" == created && "$historical" == created ]]; then
  printf 'Both requests succeeded: the suspected create-reference restriction was not reproduced.\n' >> "$GITHUB_STEP_SUMMARY"
elif [[ "$current" == created ]]; then
  printf 'Control succeeded; historical request failed. Inspect its HTTP status/body before attributing the failure to workflow permissions.\n' >> "$GITHUB_STEP_SUMMARY"
  exit 1
else
  printf 'The control failed, so the historical comparison is inconclusive. Inspect the logs for permissions or repository restrictions.\n' >> "$GITHUB_STEP_SUMMARY"
  exit 1
fi
