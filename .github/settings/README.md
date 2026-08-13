# Repository Settings

English | [日本語](README.ja.md)

This directory holds repository-level GitHub settings as JSON — the branch ruleset and the label set — so a repository derived from this boilerplate is configured reproducibly instead of by clicking through the web UI.

## These files declare intent, they do not reflect state

The JSON is the input to a one-directional apply, not a mirror of what GitHub currently enforces.

- `make apply-branch-protection` sends `branch-protection.json` as a repository ruleset named `branch-protection`, and `make create-default-labels` sends `labels.json`. Both are invoked once by `make setup-repo` when a derived repository is initialized.
- Nothing runs them again, and nothing compares them against the live state afterwards. A rule removed through the web UI, or a JSON change that was never followed by an apply, leaves declaration and reality out of step with no signal.

So this directory answers "what this repository intends to enforce", never "what this repository enforces". For the latter, ask GitHub:

```sh
gh api /repos/{owner}/{repo}/rulesets
gh api /repos/{owner}/{repo}/rulesets/{ruleset_id}
```

Both are readable with the repository's ordinary read access — on a public repository, without authenticating at all — so inspecting the live state never needs an administrative token.

## branch-protection.json

`conditions.ref_name.include` targets `production`, `staging`, `develop`, `release/**/*`, and `hotfix/**/*`. Each rule declares:

| Rule | What it declares |
| --- | --- |
| `deletion` | A targeted branch cannot be deleted. |
| `non_fast_forward` | Force-push to a targeted branch is rejected. |
| `pull_request` | Changes reach a targeted branch only through a pull request: one approving review, approvals dismissed on push, re-approval required after the last push, every review thread resolved, code-owner review, and merge restricted to merge commit or squash (rebase merge excluded). |
| `copilot_code_review` | Copilot reviews each pull request automatically, on every push and on drafts as well. |
| `code_quality` | GitHub's code quality rule blocks at `errors` severity. |
| `required_status_checks` | Seven guard-backed checks must report success before merging. |

### Applying `pull_request` to a single-maintainer repository

GitHub does not let an author approve their own pull request. With `required_approving_review_count: 1` and an empty `bypass_actors`, a repository whose only participant is its owner has nobody who can satisfy the rule, so every pull request targeting a protected branch becomes permanently unmergeable. Before applying this ruleset to such a repository, either list the maintainer in `bypass_actors`, or drop both parameters that ask for an approval — `required_approving_review_count: 0` and `require_last_push_approval: false`. Thread resolution and the merge-method restriction hold on their own.

### `code_quality` needs its backing feature to be reporting first

GitHub's own guidance is to confirm that the Code Quality workflow is running and reporting results back to pull requests **before** a ruleset declares a Code Quality threshold, because otherwise the rule can block every pull request from merging. Enabling the feature is a repository-level action outside this directory, so check it before applying rather than assuming the rule is inert where the feature is off.

### Required status checks need a reporting path on every PR

The declared required contexts are `trivy-fs-release`, `osv-release`, `trivy-config`, `sast`,
`lockfile-lint`, `openapi-security`, and `osv-diff`. Each has a `*-guard.yaml` companion that
reports the same context when the main workflow is skipped by a path or branch filter; without it,
a PR could wait forever for a check that never starts. See `docs/adr/0086-multi-layer-security-scanning.md`.

## labels.json

The label set — `name`, `description`, `color` — created by `make create-default-labels`. `make setup-repo` runs `make delete-all-labels` first, so this file is the entire intended label set rather than an addition to the labels GitHub creates by default.
