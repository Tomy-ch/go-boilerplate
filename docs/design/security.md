# Security Posture

日本語: [security.ja.md](../ja/design/security.ja.md)

This document records **how security controls in this repository are reasoned about** — what
they assume, what each one is actually for, and where each one fires. It does not enumerate
the tools; that inventory lives in [`.github/workflows/README.md`](../../.github/workflows/README.md)
and drifts with the code. What is recorded here is the part that should *not* drift.

It covers three surfaces, in the order a risk reaches them: what CI **executes** (build
inputs), what the application **links** (dependencies), and what the running service **does
with a request** (application runtime). The mechanics of identity verification are a separate
concern with their own reference — see [auth.md](auth.md); what appears here is only where
authentication sits in the enforcement model.

## Threat model

Every control below is shaped by one assumption, and it is worth stating plainly because the
shape stops making sense without it:

> **The consumers of this scaffold are internal teams whose members are known to each other.**

That assumption buys something specific: **attribution is a usable deterrent**. A control does
not have to be technically unbypassable to work, as long as a bypass is visible and traceable
to a person. The organisation supplies the consequence; the repository supplies the evidence.

It also means a whole class of attack is deliberately **out of scope**. Someone with commit
access who intends harm can delete a workflow, edit a lockfile, and change the very check that
would have caught them — all in the same pull request. No amount of CI defends against that,
and pretending otherwise produces theatre: checks that look protective, cost maintenance, and
stop nothing. Where a control cannot prevent, this repository says so and settles for
detecting.

A fork that operates under different assumptions — public contributions, untrusted committers,
regulatory obligation — should revisit this document first, then the controls. Several of them
would need to become fail-closed.

## Three kinds of control

Conflating these is the most common way a security setup becomes both annoying and useless.
Each mechanism here is exactly one of the three, on purpose.

| Kind | What it does | Failure mode when misapplied |
| --- | --- | --- |
| **Enforcement** | Refuses the action. Fail-closed. | Blocks legitimate emergency work; gets disabled |
| **Detection** | Lets the action through, records it, makes it attributable | Mistaken for protection; nobody reads the record |
| **Deterrence** | Organisational consequence, informed by detection | Assumed to exist without evidence to act on |

Where each mechanism sits:

| Mechanism | Kind | Note |
| --- | --- | --- |
| `pin-actions-check` / `pin-images-check` | Enforcement | Fail-closed: an unpinned or unregistered reference is an error |
| Release gates (`trivy-release-gate` / `osv-release-gate`) | Enforcement | Only on PRs into a deploy branch |
| `dependency-review` | Enforcement | Evaluates only what the PR *adds* |
| Secret scans (gitleaks / TruffleHog) | Enforcement | A committed secret is never an acceptable trade |
| `zizmor` | Enforcement | High severity only; exceptions are file-scoped in `.github/zizmor.yml` |
| Reporting scanners (`trivy-fs` / `osv-scanner` / `govulncheck` / CodeQL) | Detection | Findings reach code scanning and the PR, but do not block |
| `npm-cooldown-audit` | Detection | Non-blocking by construction — see below |
| `harden-runner` (`egress-policy: audit`) | Detection | Records egress; does not restrict it |
| `CODEOWNERS` | Enforcement | The review requirement behind "this decision belongs to a role" |
| OpenSSF Scorecard | Detection | Posture measurement, no verdict |

## Where a control fires

A control that runs everywhere is a control nobody reads. Each trigger answers a different
question, and a mechanism belongs where its question is actually being asked:

| Trigger | Question it answers |
| --- | --- |
| Pull request | *Does this change make things worse?* |
| Push to a protected branch | Keeps a code-scanning baseline for branch protection to judge |
| PR into `develop` / `staging` / `production` | *Is what we are about to promote acceptable?* |
| Weekly schedule | *Did the world change while the code stood still?* — new CVEs, new queries, newly archived actions |

The consequence worth internalising: **a scheduled run only exists for a tool whose result can
change without the code changing.** Everything else is noise on a timer.

### Why reporting and gating are split

An ordinary PR reports; a promotion PR gates. The reasoning is in
[ADR-0080](../adr/0080-multi-layer-security-scanning.md), and reduces to this: a vulnerability
inherited from the existing dependency tree is not something the current PR introduced, and
not something its author can fix. Blocking there turns every unrelated change red until an
upstream fix lands elsewhere. The predictable outcome is that the check gets disabled or
routinely merged past — which costs more than the gate was worth.

Suppressing the finding instead (`ignore-unfixed`, an ID allowlist) trades one failure for
another: it also silences the finding at promotion time, which is the one moment it mattered.
The split keeps the signal in both places and puts the *verdict* where someone can act on it.

## Build inputs

What CI *executes* is a different risk from what the application *links*. Actions and base
images run with the job's credentials before any of our code does, so they are pinned harder.

**A version reference is not an identity.** A tag can be re-pointed and a mutable image tag can
be rebuilt, so the version stays in the source as human-readable intent while an immutable
digest lives in a lockfile that is the single source of truth —
`.github/actions-pin.toml` ([ADR-0081](../adr/0081-sha-pinned-actions.md)) and
`docker/images-pin.toml`. Both checks are fail-closed, and unpinned versus unregistered are
distinct errors so neither degrades into the other.

## Dependencies

### Two principles that hold for every ecosystem

**A cooldown window is proportional to upstream's detection latency, not to blast radius.** A
compromised publish is usually caught and pulled within hours to days; the window exists to sit
out that interval. Where upstream detects and corrects quickly the window is shorter (npm: 7
days), and where remediation is slower it is longer (Actions and images: 14 days,
`PIN_ACTIONS_MIN_AGE_DAYS` / `PIN_IMAGES_MIN_AGE_DAYS`). The number is a policy input derived
from the ecosystem, not a constant to copy across them.

**The window is a proxy, and a proxy can be discharged by direct evidence.** Waiting N days is a
cheap stand-in for four questions: did the publisher change, does the artifact match its source,
what actually changed, and did new dependencies appear. When a fix is urgent those can be
answered directly — transparency-log lookup, artifact-versus-tag comparison, reading the diff,
checking the manifest for new requirements. Answering them beats counting days. Skipping *both*
does not.

### Go modules

The Go ecosystem gives integrity almost for free, and offers nothing at all for freshness. Both
halves matter.

**Integrity is strong and on by default.** Every module version's hash is recorded in a public
transparency log (`sum.golang.org`) and checked against `go.sum` on download; a published
version is immutable in the proxy and cannot be withdrawn or silently swapped. Nothing in this
repository weakens that — there is no `GOFLAGS`, `GONOSUMDB`, `GOPRIVATE`, or `GOINSECURE`
override anywhere. `vendor/` is untracked, so build inputs are re-fetched and re-verified rather
than trusted from the tree.

**Freshness has no enforcement whatsoever.** There is no `min-release-age` equivalent: `go get`
will take a version published minutes ago without complaint. This inverts the npm situation in a
way worth internalising:

| | npm | Go modules |
| --- | --- | --- |
| Cooldown enforced by the toolchain | Yes, at resolution | **No** |
| Can be bypassed | Yes, with a flag | Nothing to bypass |
| Bypass is detectable | Yes (`npm-cooldown-audit`) | **Nothing to detect** |
| Remaining control | Review + detection | **Review only** |

Because review is the only control here, `go.mod` and `go.sum` are in
[`CODEOWNERS`](../../.github/CODEOWNERS). That entry is load-bearing rather than symmetrical
bookkeeping: for npm a slip is caught after the fact, for Go it is not caught at all.

**Reachability filtering cuts both ways.** `govulncheck` reports only vulnerabilities the
application actually calls, which is what makes it trustworthy enough to act on. The cost is
coverage: an advisory the Go vulnerability database has not ingested yet produces no finding at
all, so a clean `govulncheck` says nothing about a GHSA published this week. Breadth is covered
separately by Trivy FS and OSV-Scanner, which match on version rather than call graph.

### npm

The cooldown is real but narrower than it looks: `min-release-age` is enforced while npm
**resolves**, and `npm ci` does not resolve — it replays the lockfile. Every CI job and image
build here uses `npm ci`.

Verified behaviour, not inference:

| Action | Result |
| --- | --- |
| Resolving an in-window version | Rejected (`ETARGET`) |
| Resolving a range whose newest match is in-window | Silently resolves to the newest *aged* version, exit 0 |
| Resolving with `--min-release-age=0` | Succeeds; the in-window version enters the lockfile |
| `npm ci` with an in-window entry already in the lockfile | Succeeds, no warning |
| `npm install` / adding a package, with that entry present | Succeeds; the entry is kept |

Two consequences. A deliberate override costs the team nothing — no member is blocked and no
workaround is needed. And it leaves no trace anywhere: not at image build, not in CI, and once
the window passes the entry is indistinguishable from a normally resolved one.

That is why `npm-cooldown-audit` exists, why it audits a **PR against its base** rather than only
scanning the current tree (an entry ages out of the window; the PR comment does not), and why it
**never fails the build**. Overriding the cooldown is a role's call — reacting to a CRITICAL
advisory is the case it exists for — so a hard gate would block precisely the legitimate use. The
non-blocking property is implemented in the tool itself rather than in workflow configuration, so
it cannot be turned into a gate by editing YAML.

**A transitive pin is provisional debt.** Forcing a patched version through `overrides` is
written as a same-major floor (`">=<fixed> <<next-major>"`), never an exact version: an exact pin
freezes the dependency where it is, so when the pinned version later gets its own advisory the
pin keeps forcing the now-vulnerable one. Every override is meant to be reclaimed once the parent
ships a release that pulls the fix natively.

## Application runtime

The controls above protect what enters the repository. These protect what the running service
does with a request. Three patterns repeat, and they are the part worth carrying into a fork.

**Deny by default, at every boundary.** The outbound dial guard
(`internal/observability/http_client_transport.go`, [ADR-0020](../adr/0020-egress-ssrf-guard.md))
refuses link-local, multicast, unspecified, and bogon destinations unconditionally, and refuses
loopback / private / CGNAT unless the caller opted in through the context — with the unset case
resolving to the safe `false`. Error-detail exposure
(`internal/controller/httpstack/errorhandler/detail_exposure.go`) is the same shape: route
mismatch, unresolved operation, empty `operationId`, and not-opted-in all evaluate to *deny*.
Neither control has a state where forgetting something opens it.

**The spec is the authority for the request boundary.** Request validation and authentication are
enforced at runtime from the OpenAPI document ([ADR-0013](../adr/0013-spec-driven-request-validation.md)),
not from hand-written checks in handlers. The direct consequence is that **reviewing the spec
diff is reviewing the security posture** — an operation that omits its `security` requirement is
unprotected no matter how the handler is written, and no amount of Go review will surface it.
Business-validity rules are deliberately *not* here: the domain layer is their sole authority
([ADR-0014](../adr/0014-validation-value-authority.md)), so the two never drift into each other.

**Escape hatches are named, narrow, and greppable.** `ContextWithAllowPrivateNetwork`, the
`details` property that opts an error schema into detail exposure, and `/metrics` as a declared
auth exception ([ADR-0016](../adr/0016-metrics-endpoint-auth-exception.md)) are each a specific
seam you can search for and enumerate. None of them is a general-purpose flag, because a general
flag becomes the thing everyone sets.

Two absences are deliberate rather than pending: there is no in-application rate limiter
([ADR-0094](../adr/0094-no-in-app-rate-limiter.md)), and responses are not validated against the
spec ([ADR-0013](../adr/0013-spec-driven-request-validation.md)). SQL injection is handled
structurally instead of by review — queries are generated by sqlc and therefore parameterised
([ADR-0022](../adr/0022-sqlc-type-safe-sql.md)) — and `gosec` runs in the authoritative
golangci gate ([ADR-0075](../adr/0075-two-layer-golangci-config.md)).

A detail worth copying rather than rediscovering: Go's `netip.Addr.IsPrivate` covers RFC1918 and
ULA but **not** CGNAT (`100.64.0.0/10`), so the guard carries its own prefix check. A fork that
reimplements the dial guard from the standard library alone inherits that hole.

## Secrets

Two detectors, because a secret scanner can be wrong along two independent axes: it can fail to
recognise the shape, or it can flag a string that was never a live credential. gitleaks covers
pattern-shaped detection over the full history; TruffleHog reports only credentials it verified
against the issuing service.

One rule overrides convenience everywhere: **a detected secret's value never reaches a job log,
a PR comment, or an artifact.** Summaries carry detector name, path, line, and commit — never
the match itself. A leak report that leaks is worse than no report, because it publishes the
credential to a wider audience than the commit did.

## Honest limits

Worth stating so nobody builds on a stronger assumption than the controls support:

- `harden-runner` runs in `audit` mode. It **records** egress; it does not restrict it. Moving
  to `block` requires a settled endpoint allowlist, which is deliberately deferred until the
  audit data exists.
- Release gates only block if registered as **required status checks**. Without that branch
  protection rule they report and nothing more, and the reporting/gating split degrades to
  reporting everywhere.
- `CODEOWNERS` only enforces under the **"Require review from Code Owners"** rule; otherwise it
  merely auto-requests a reviewer.
- Detection reaches a PR comment and a run annotation. Both are passive, and the annotation
  disappears with the run's retention. Routing findings to a channel the owning role actually
  watches is tracked separately.

## Related

- [ADR-0080](../adr/0080-multi-layer-security-scanning.md) — layered scanning, reporting/gating split, runner hardening
- [ADR-0081](../adr/0081-sha-pinned-actions.md) — SHA-pinned Actions and the supply-chain quarantine
- [ADR-0091](../adr/0091-release-image-supply-chain.md) — release-image integrity (signing, provenance, SBOM)
- [ADR-0075](../adr/0075-two-layer-golangci-config.md) — `gosec` as an in-process check during static analysis
- [`.github/workflows/README.md`](../../.github/workflows/README.md) — the workflow inventory and full trigger matrix
