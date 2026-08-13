# Evidence collection — Go modules

Read this together with the axis definitions in `SKILL.md`. Go inverts the usual balance: integrity
is nearly free and freshness has no enforcement at all (`docs/design/security.md` → "Go modules").
That shapes which axes are cheap here and which carry the weight.

Two consequences to hold onto:

- **There are no lifecycle scripts.** `go get` does not run code from the module. The install-hook
  attack class does not exist, so axis D's attention belongs on code that runs *later*: `init()`
  functions, `//go:embed` blobs, `go:generate` directives, and build-constraint-hidden files.
- **`go.mod` / `go.sum` are in `CODEOWNERS` precisely because review is the only freshness control.**
  A triage here is not a supplement to a toolchain gate — it *is* the gate.

Set up:

```sh
MOD=<module-path>      # e.g. github.com/owner/repo
CAND=<candidate>       # e.g. v1.9.0
BASE=<baseline>        # the version in go.mod
WORK="$SCRATCH/go-$(echo "$MOD" | tr / -)"; mkdir -p "$WORK"; cd "$WORK"
go list -m -versions "$MOD"
curl -fsSL "https://proxy.golang.org/$(echo "$MOD" | tr 'A-Z' 'a-z')/@v/$CAND.info"
```

## Axis P — publisher

A module has no publisher account; the VCS history is the publisher record.

```sh
git clone --filter=blob:none --no-checkout "https://$MOD" repo 2>/dev/null; cd repo
git log --format='%an <%ae> %d' "$BASE".."$CAND" | sort -u
git log --format='%an' "$BASE" | sort | uniq -c | sort -rn | head   # who normally commits here
gh api "repos/<owner>/<repo>" -q '{owner: .owner.login, archived, fork, pushed_at}'
cd "$WORK"
```

- A committer in the candidate range who appears nowhere in the project's history is the P finding.
- Check for **ownership change or archival** — a transferred or archived repo whose module path
  still resolves is a takeover-shaped situation.
- A tag pointing at a commit that is not an ancestor of the default branch belongs to axis A below.

## Axis A — artifact versus source

Mostly answered for free, but answered *narrowly* — read the limit before scoring `0`.

```sh
# the proxy's recorded hashes; sum.golang.org is consulted automatically on download
go mod download -x -json "$MOD@$CAND" | jq '{Version, Info, Zip, Sum, GoModSum}'
```

- Inclusion in `sum.golang.org` plus a matching `go.sum` proves the artifact **was not swapped after
  publication**, and that the version is immutable in the proxy. Confirm nothing in the environment
  weakens it: `go env GOFLAGS GONOSUMDB GOPRIVATE GOINSECURE GONOSUMCHECK` should be empty, and this
  repo sets none.
- **This does not prove benignity.** A maliciously published version is verifiably, immutably
  malicious. Score `0` for "not tampered post-publish", and let axis D carry the actual question.
- The sharper A question is whether the module zip matches the VCS tag. Compare them:

  ```sh
  unzip -q "$(go env GOMODCACHE)/cache/download/$(echo "$MOD" | tr 'A-Z' 'a-z')/@v/$CAND.zip" -d zip
  git -C repo checkout -q "$CAND" 2>/dev/null || git -C repo checkout -q "v${CAND#v}"
  diff -rq "zip/$MOD@$CAND" repo --exclude='.git' | grep -v '^Only in repo' | head -30
  ```

  A file in the zip with no counterpart at the tag is a `3`. If the tag is missing or the repo is
  unreachable, A is `?` — the transparency log alone does not answer this.

## Axis D — what actually changed

Read the VCS diff rather than the zip diff; it is the same content with authorship and messages
attached.

```sh
cd repo && git log --oneline "$BASE".."$CAND" | head -40
git diff --stat "$BASE".."$CAND" | tail -20
git diff "$BASE".."$CAND" | head -500
```

Then grep the range specifically for the Go-shaped signatures:

```sh
git diff "$BASE".."$CAND" -U0 | grep -nE '^\+.*(os/exec|exec\.Command|net\.Dial|http\.(Get|Post|NewRequest)|os\.Getenv|os\.ReadFile\("/|unsafe\.|//go:embed|//go:generate|syscall\.|import "C")' | head -40
git diff "$BASE".."$CAND" --name-only | grep -E '_(linux|darwin|windows)\.go$|^\.github/' | head
git diff "$BASE".."$CAND" -U0 | grep -nE '^\+func init\(' | head
cd "$WORK"
```

What each one is for:

- **`func init()` additions** — the only place module code runs merely because it was imported. A new
  `init()` in a library that had none deserves reading in full.
- **`os/exec`, `syscall`, `import "C"`** in a package with no reason to spawn or descend to cgo.
- **`//go:embed` of a new blob**, or a new binary file in the tree — the payload carrier when source
  must stay readable.
- **`//go:generate`** — runs during development, with the developer's credentials.
- **Platform-suffixed or build-tagged files**, which compile only on some targets and are the easiest
  place to hide code from a casual reviewer.
- **`.github/` changes inside the released range** — a workflow the release process itself runs.

Vendored trees are a size trap: `vendor/` churn dominates `--stat`. Filter it out for the read
(`git diff "$BASE".."$CAND" -- . ':(exclude)vendor'`) but do not ignore it entirely — an edited
vendored file that no upstream bump explains is a finding.

## Axis S — dependency and capability surface

```sh
diff <(go mod download -json "$MOD@$BASE" | jq -r .GoMod | xargs cat 2>/dev/null) /dev/null >/dev/null 2>&1
cd repo && git diff "$BASE".."$CAND" -- go.mod go.sum | head -60; cd "$WORK"
```

For each newly required module, check what it is: a package created recently, with one tag and no
history, pulled in by a mature dependency, is the same smuggling pattern npm sees. Also note whether
the candidate adds a `main` package or a `cmd/` binary the repo would then build.

Finally, run the reachability question the repo already trusts — but read its limit:

```sh
govulncheck ./...
```

A clean result means no *known* advisory is reachable. `docs/design/security.md` records the cost:
an advisory the Go vulnerability database has not ingested produces no finding at all, so this says
nothing about a publish from this week — which is exactly the situation triage is in. Use it as
corroboration, never as the answer.
