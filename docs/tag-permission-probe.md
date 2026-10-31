# Probe historical tag permissions

This isolated experiment checks the conditional token-permission finding in [Omnigrex PR #161](https://github.com/jozala/omnigrex/pull/161).
It uses the workflow's normal `GITHUB_TOKEN` with exactly `contents: write` and `actions: read`, not a PAT or GitHub App credential.
It calls `POST /repos/jozala/omnigrex-testrun/git/refs`, matching the release implementation's lightweight tag creation.
It does not build images, create GitHub Releases, change repository settings, or access any deployment.

## Run

1. Open **Actions -> Probe historical tag permissions -> Run workflow**.
2. Select branch **main** and use the supplied historical commit default.
3. Run the workflow and inspect the job summary and the HTTP responses in **Compare current and historical tag creation**.

The control tags the dispatched main tip.
The second request tags an older main ancestor with different workflow contents and no branch head or tag pointing directly to it.
The workflow validates those preconditions before issuing either request.
Both requests run even when one fails, so the comparison is visible.

## Interpret

| Control | Historical | Interpretation |
| --- | --- | --- |
| Created | Created | The suspected restriction was not reproduced; historical reports alone should not keep the finding blocking. |
| Created | Failed | Inspect the historical HTTP response; a workflow-related 403 supports the finding. |
| Failed | Either | Inconclusive; investigate general permissions or repository rules first. |

A precondition failure is not evidence of a GitHub token limitation.
A failed historical request makes the job red, which may be the expected result of this experiment rather than a broken probe.
The job's **Set up job** output shows the actual token permissions.
Do not paste credentials or enable token/debug tracing to diagnose a failure.

## Cleanup and repetition

Successful requests leave uniquely named tags for inspection: `permission-probe-RUN_ID-ATTEMPT-current` and `permission-probe-RUN_ID-ATTEMPT-historical`.
The job summary lists the exact names and outcomes.
Delete only those successfully created probe tags before repeating the same historical target; do not delete existing application branches or unrelated tags.
A retained historical probe tag changes GitHub's existing-ref condition, so the next run intentionally refuses to proceed until cleanup.
No automated cleanup or tag overwrite is performed.

## Limits

The repository is public, has no branch protection or effective main-branch rules, and had no workflows or tags when this experiment was prepared.
Those facts were checked during setup and may change later.
This probe tests tag creation only, not the full release-publication workflow or anonymous image access.
If tag creation succeeds, release creation and alpha-to-stable promotion still need their separately authorized acceptance tests.
