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

Both are readable with ordinary repository read access on a public repository; a private one needs the `administration: read` permission.

## branch-protection.json

`conditions.ref_name.include` targets `production`, `staging`, `develop`, `release/**/*`, and `hotfix/**/*`. Each rule declares:

| Rule | What it declares |
| --- | --- |
| `deletion` | A targeted branch cannot be deleted. |
| `non_fast_forward` | Force-push to a targeted branch is rejected. |
| `pull_request` | Changes reach a targeted branch only through a pull request: one approving review, approvals dismissed on push, re-approval required after the last push, every review thread resolved, and merge restricted to merge commit or squash (rebase merge excluded). |
| `copilot_code_review` | Copilot reviews each pull request automatically, on every push and on drafts as well. |
| `code_quality` | GitHub's code quality rule blocks at `errors` severity. |

### Applying `pull_request` to a single-maintainer repository

GitHub does not let an author approve their own pull request. With `required_approving_review_count: 1` and an empty `bypass_actors`, a repository whose only participant is its owner has nobody who can satisfy the rule, so every pull request targeting a protected branch becomes permanently unmergeable. Before applying this ruleset to such a repository, either list the maintainer in `bypass_actors`, or set the count to `0` and keep the remaining parameters — thread resolution and the merge-method restriction still hold on their own.

### CI results do not block merge

No `required_status_checks` rule is declared, so no workflow result gates a merge; a pull request can be merged over a red check.

Adding one is not just a matter of listing check contexts. A required check whose workflow is `paths`-filtered never reports on a pull request that skips it, which blocks the merge forever — so each gate registered as required needs a `*-guard.yaml` companion reporting the same check context on the complementary path set. `docs/adr/0080-multi-layer-security-scanning.md` records that design.

## labels.json

The label set — `name`, `description`, `color` — created by `make create-default-labels`. `make setup-repo` runs `make delete-all-labels` first, so this file is the entire intended label set rather than an addition to the labels GitHub creates by default.
