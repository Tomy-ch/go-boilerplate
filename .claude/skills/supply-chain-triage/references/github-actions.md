# Evidence collection — GitHub Actions

Read this together with the axis definitions in `SKILL.md`. This is the ecosystem where all four
axes are answerable, because the artifact *is* a commit range in a public repository — there is a
real diff, real authorship, and a checkable tag-to-commit relationship. It is also the highest
exposure class: an action runs with the job's `GITHUB_TOKEN` and whatever `secrets` the step
receives, before any first-party code executes.

Set up:

```sh
ACT=<owner/repo>       # e.g. actions/checkout
CAND=<candidate tag>   # the tag the skill wants to pin, e.g. v5.1.0 or v5
BASE_SHA=<baseline>    # the SHA currently in .github/actions-pin.toml
export GITHUB_TOKEN="$(gh auth token)"
WORK="$SCRATCH/gha-$(echo "$ACT" | tr / -)"; mkdir -p "$WORK"; cd "$WORK"
```

The baseline is the **SHA in the lockfile**, not the previous tag. That is what the workflows run
today, and the lockfile is the repo's source of truth for it:

```sh
grep -n "$ACT" .github/actions-pin.toml
```

## Axis A first — the tag-repointing check

Do this axis before the others here, because it is cheap and it is how the real incidents were
detected. A Git tag is mutable; re-pointing an already-trusted tag at a malicious commit is the
attack that hit `tj-actions/changed-files` (2025), and the lockfile is what makes it visible.

```sh
# what the tag resolves to now (annotated tags: ^{} is the dereferenced commit)
git ls-remote "https://github.com/$ACT" "refs/tags/$CAND" "refs/tags/$CAND^{}"

# is that commit on the default branch?
NEW_SHA=<from above>
gh api "repos/$ACT/compare/$(gh api repos/$ACT -q .default_branch)...$NEW_SHA" -q '.status'
```

Score `3` immediately on either of these:

- **A tag already recorded in `.github/actions-pin.toml` now resolves to a different SHA.** The
  version reference did not change but the code did. That is not an upgrade, it is a substitution.
- **The tag's commit is not reachable from the default branch** (`compare` returns `diverged`, or the
  commit is not an ancestor). Releases are cut from the branch; a tag hanging off it is either a
  release-process anomaly worth explaining or the attack.

Then check whether the release itself is corroborated, and by whom:

```sh
gh api "repos/$ACT/releases/tags/$CAND" -q '{tag: .tag_name, author: .author.login, created: .created_at, published: .published_at, draft, prerelease}'
gh api "repos/$ACT/tags" -q '.[] | select(.name=="'"$CAND"'") | .commit.sha'
```

A release whose `author` differs from the historical release author, or whose `created_at` and
`published_at` are far apart, is a P observation — carry it down.

## Axis P — publisher

```sh
gh api "repos/$ACT" -q '{owner: .owner.login, archived, fork, disabled, pushed_at, default_branch}'
gh api "repos/$ACT/compare/$BASE_SHA...$NEW_SHA" \
  -q '.commits[] | "\(.sha[0:8]) \(.author.login // "?") / \(.commit.author.name) \(.commit.verification.verified)"'
gh api "repos/$ACT/commits?per_page=100" -q '[.[].author.login] | group_by(.) | map({(.[0]): length}) | add'
```

- A `login` in the candidate range that does not appear in the repo's recent history is the finding.
- `verification.verified` false across a repo that otherwise signs commits is a `2`.
- An **archived or newly transferred** repo whose action is still referenced is a standing risk worth
  reporting regardless of this version's score.

## Axis D — what actually changed

The comparison is a real commit range. Read all of it; these diffs are small.

```sh
gh api "repos/$ACT/compare/$BASE_SHA...$NEW_SHA" -q '.files[] | "\(.status)\t\(.changes)\t\(.filename)"'
gh api "repos/$ACT/compare/$BASE_SHA...$NEW_SHA" -q '.commits[].commit.message' | head -20
```

Fetch the patch itself for the files that matter:

```sh
gh api "repos/$ACT/compare/$BASE_SHA...$NEW_SHA" \
  -H 'Accept: application/vnd.github.v3.diff' > range.diff
grep -nE '^\+.*(process\.env|ACTIONS_RUNTIME_TOKEN|GITHUB_TOKEN|child_process|execSync|eval\(|atob\(|curl |wget |base64 -d|/proc/self|memdump|https?://[0-9]+\.)' range.diff | head -40
```

Three action shapes, three places the payload lives:

- **JavaScript action** (`runs.using: node*`): the committed bundle under `dist/` is what runs; `src/`
  is what gets reviewed. **A `dist/` change with no corresponding `src/` change is the single
  highest-signal finding in this ecosystem — score `3`.** Conversely, verify the bundle is consistent
  with the source rather than assuming it:

  ```sh
  awk '/^diff --git/{f=$0} /^\+/{if (f ~ /dist\//) d++; if (f ~ /src\//) s++} END{print "dist+:", d+0, "src+:", s+0}' range.diff
  ```

  A large `dist+` against a zero or near-zero `src+` is the shape to escalate on.

- **Composite action** (`runs.using: composite`): read every added `run:` step in `action.yml`. <!-- skill-lint-ignore -->
  A new shell step is arbitrary code execution with the job's credentials.

- **Docker action** (`runs.using: docker`): a changed `image:` or `Dockerfile` moves the analysis to
  `references/docker-images.md`, with the same thin-evidence caveats.

Also read any `.github/workflows/` change inside the range: `pull_request_target`, a new `secrets`
reference, or a self-hosted runner in the upstream repo affects how *their* releases are produced,
which is the pipeline you are trusting.

## Axis S — dependency and capability surface

```sh
gh api "repos/$ACT/contents/action.yml?ref=$NEW_SHA" -q '.content' | base64 -d > action.new.yml
gh api "repos/$ACT/contents/action.yml?ref=$BASE_SHA" -q '.content' | base64 -d > action.old.yml
diff action.old.yml action.new.yml
```

- New `inputs` are usually benign; a new **default that points at a URL or a token** is not.
- A JS action's `package.json` gaining dependencies means the bundle now carries code from somewhere
  else — check the new dependency the way `references/npm.md` does.
- Note any added network endpoint. This repo runs `harden-runner` in `audit` mode, so egress is
  recorded but not restricted (`docs/design/security.md` → "Honest limits") — a new endpoint will be
  visible after the fact, not blocked.

## Reporting notes

- The exposure line is always the highest class: **executes in CI with the job's credentials**.
- When `/actions-pin` reached this skill via its step-back rule, say which alternative exists — the
  newest aged exact version in the same major is usually adoptable now and needs no override at all.
  A HIGH score here has a cheap answer.
- If axis A found a re-pointed tag, that is an upstream security event, not a version decision. Name
  the old and new SHAs and the affected workflow files (`grep -rn "$ACT" .github/`) so it can be
  reported and the current pin audited.
