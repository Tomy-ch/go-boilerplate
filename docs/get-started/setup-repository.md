# Post-Repository Clone Task List

English | [日本語](../ja/get-started/setup-repository.ja.md)

For details of Make commands, refer to [Make Target List](../../.makefiles/README.md).

## Phase 1: Install mise and activate it in your shell

This project requires [mise](https://mise.jdx.dev) as the tool / runtime version manager. Install it via the [official installation guide](https://mise.jdx.dev/getting-started.html), then **activate it in your shell init** — this is mandatory, not optional. The repository's Make targets resolve `golangci-lint`, `lefthook`, etc. through mise's shims, and the shims are only on `PATH` after activation:

```sh
# zsh
echo 'eval "$(mise activate zsh)"' >> ~/.zshrc

# bash
echo 'eval "$(mise activate bash)"' >> ~/.bashrc

# then reload (or open a new terminal)
exec $SHELL
```

Verify activation with:

```sh
mise --version
which mise
```

## Phase 2: Install the Go runtime and project tools

All tool versions (golangci-lint / sqlc / oapi-codegen / mockgen / dlv / lefthook / ...) are pinned in [`mise.toml`](../../mise.toml) as the single source of truth. The Dockerfiles, the local installer (`.makefiles/go/installer.mk`), and the CI workflows all install only what they need via `mise install <tool>` against the same `mise.toml`.

```sh
make go-update       # installs the pinned Go runtime
make install-tools   # installs gopls / gotests / impl / dlv / lefthook / golangci-lint / zizmor
make activate-tools  # runs `lefthook install` to wire git hooks
```

## Phase 3: Install the agent configuration (recommended)

The AI-assist layer ships as configuration: project-scoped official plugins, this repository's own skills under [`.claude/`](../../.claude/README.md) / [`.codex/`](../../.codex/README.md), and one officially recommended external skill (`graphify`, a queryable knowledge graph of the repository). Two idempotent bootstraps install the parts a clone does not carry:

```sh
bash .claude/scripts/bootstrap-plugins.sh          # official plugins (project scope)
bash .claude/scripts/bootstrap-external-skills.sh  # external skills (user scope: Claude Code + Codex)
```

`graphify` itself is pinned in `mise.toml` like every other tool, so `mise install` already fetched it; the bootstrap only writes the skill into each assistant's config directory. Which of its commands stay local and which reach an LLM API is documented in [`.claude/README.md`](../../.claude/README.md).

**Declining the AI-assist layer is the adopting architect's call.** This template is built to stay fully maintainable without AI tooling — the layering rules live in [docs/rules.md](../rules.md), not in the assistant configuration — so nothing above is load-bearing for building, testing, or shipping. A fork that does not want the layer should remove it deliberately rather than leave it half-configured:

- skip both bootstraps; no other phase of this setup depends on them, and
- drop what you do not want to carry: `.claude/`, `.codex/`, the `pipx:graphifyy[sql]` pin in `mise.toml`, `.graphifyignore`, and the `graphify-out/` entries in `.gitignore`, `.markdownlint-cli2.yaml`, and `scripts/mermaid-lint/index.ts`.

Removing it later costs the same as removing it now, so adopting the recommended configuration first and deciding afterwards is a safe order.

## Phase 4: Local Startup Verification

Start the application locally and confirm it works without issues.

```sh
make serve
make tools
make db-init
```

<!-- boilerplate-only:begin -->
## Phase 5: Execute Localization Script

Run the following commands to execute the script that replaces the Go module name in bulk.

Replace ORG, REPO, and CODE_OWNERS as appropriate. Only change derived settings if necessary.

```sh
export ORG=<your-org/git-user-name>
export REPO=<your-repo>

# CODEOWNERS owner — a user (@name) or a team (@org/team). An organization itself
# cannot own a path, so a fork owned by one must name a team.
export CODE_OWNERS=<@your-org/tech-leads>

export MODULE=${REPO}
export APP_NAME=${REPO}
export OPENAPI_TITLE=${REPO}
export COPILOT_TITLE=${REPO}
export COPYRIGHT_HOLDER=${ORG}
export COPYRIGHT_YEAR=$(date +%Y)

make setup-replace-module OLD_MODULE=go-boilerplate NEW_MODULE=$MODULE
make setup-replace-repository-reference REPOSITORY=$ORG/$REPO
make setup-replace-app-metadata APP_NAME=$APP_NAME OPENAPI_TITLE="$OPENAPI_TITLE" COPILOT_TITLE="$COPILOT_TITLE"
make setup-replace-license-copyright COPYRIGHT_HOLDER="$COPYRIGHT_HOLDER" COPYRIGHT_YEAR=$COPYRIGHT_YEAR
make setup-replace-codeowners OWNERS="$CODE_OWNERS"
make gen-api
make gen-sqlc
make tidy-lib

# Verify the replacements landed, then remove the localization tooling.
# Run this last: the scripts above are one-shot, and keeping them lets you re-run
# any of them until this passes.
make setup-verify
```

`setup-verify` checks that every file `replace-module` claims to cover is free of the
boilerplate name, and that LICENSE, CODEOWNERS, README and the OpenAPI spec carry the values you
passed. Only once it passes does it delete `scripts/setup/replace-*` and itself — re-applying
them to an already-localized repository is wrong, since `replace-codeowners` rewrites *every*
rule's owner to a single value. `scripts/setup/lib` goes with whichever of the two one-shot
tools (this one and the sample remover) runs last.

## Phase 6: Localization Verification

Confirm that basic functionality works correctly, including tests, static analysis, code generation, and health checks.

```sh
make test
make lint
make gen
curl http://localhost:8080/health
curl http://localhost:8080/ready
```

## Phase 7: Manual Rewrites

1. Rewrite the contents of README.md according to your project.
2. Rewrite the contents of [openapi.yaml](../../openapi/openapi.yaml) according to your project.
    - Rewrite the entire Info section according to your project.
        - title
        - termsOfService
        - contact
        - version
        - description
        - license

<!-- boilerplate-only:end -->
## Phase 8: Rewrite env Files

Rewrite the files in the [env/](../../env/) directory according to your project.

For the meaning of configuration values, refer to [env/README.md](../../env/README.md).

### Review the clamped config values

A few subsystems **clamp** out-of-range values to safe defaults instead of failing startup — a resilience choice so a misconfigured process still runs. A clamp only triggers when you explicitly set `0` / a negative / an invalid value (the shipped env vars carry sane `envDefault`s), but because the correction is applied silently at runtime, review these when you tune the corresponding `WORKER_*` / `OUTBOX_*` / `SECURE_COOKIE_*` values:

- **Worker engine** (`WORKER_*`) — `Settings.normalize()`; see [internal/controller/worker/README.md](../../internal/controller/worker/README.md) (Config clamping).
- **Outbox relay** (`OUTBOX_*`) — `provideRelaySettings`; see [internal/controller/outbox/README.md](../../internal/controller/outbox/README.md) (Settings → Clamping).
- **Secure cookie** (`SECURE_COOKIE_SAME_SITE`) — `normalizeSameSite` clamps any non-`Lax`/`Strict`/`None` value to "do not override"; see [internal/controller/httpstack/cookie/README.md](../../internal/controller/httpstack/cookie/README.md).

## Phase 9: Repository Initialization

After completing the above steps, initialize the repository after the first push.

### When starting from a GitHub template

```sh
git add -A
git commit -m "Initial commit: setup boilerplate for $REPO"
git push origin main
make setup-repo
make branch-minor
```

### When starting from Git Clone

```sh
git remote set-url origin ${ORG}/${REPO}
git add -A
git commit -m "Initial commit: setup boilerplate for $REPO"
git push -u origin main
make setup-repo
make branch-minor
```

## Phase 10: Create Deployment Configuration

This boilerplate adopts a configuration that does not depend on a specific cloud provider or deployment method, allowing flexible deployment to various cloud or on-premise environments.

Therefore, deployment settings do not include a specific deployment target. Add necessary settings according to your project's deployment destination.

Deployment CI/CD: Complete [.github/workflows/deploy-app.yaml](../../.github/workflows/deploy-app.yaml).

`Note: Please modify this section according to your environment` indicates sections that need to be modified according to your environment.

## Phase 11: Implement Authentication & Authorization

This boilerplate ships **development-only stubs** for both authentication (authn) and authorization (authz), and they are wired **only** for the `local` / `ci` / `test` environments. For `development` / `staging` / `production` the DI providers are **fail-closed**: they refuse to wire the stub and return an error, so the application **deliberately fails to start** until you implement and wire real components.

This is an intentional forcing function — it guarantees a signature-skipping authenticator or an allow-all authorizer can never ship to a real environment. **Implementing both for `development` / `staging` / `production` is a required project-start task.**

> [!IMPORTANT]
> The `Authorizer` is provided inside `InfrastructureModule`, so **every process that builds a usecase** — the HTTP server **and** the background job / worker processes — requires a configured `Authorizer`. Until the authorization step below is done, running any of them with `APP_ENV=development` / `staging` / `production` exits at Fx construction with `no authorizer configured for environment` (authn behaves the same with `no authenticator configured for environment`). Seeing this before you implement the real components is expected, not a bug.

### Authentication (authn)

This boilerplate includes sample code using JWT as an example implementation of authentication. Implement authentication according to your project requirements.

Create authentication functionality by implementing the usecase [Authenticator](../../internal/usecase/boundary/auth/authenticator.go) interface.

- Reference: [internal/infrastructure/auth/README.md](../../internal/infrastructure/auth/README.md)
- Stub example (local, signature-less): [internal/infrastructure/auth/local/auth_local.go](../../internal/infrastructure/auth/local/auth_local.go)
- Add your `stg` / `prd` implementations (JWT / OAuth2 / OIDC / Cognito / Auth0 など) under `internal/infrastructure/auth/{stg,prd}/`.
- Wire them per environment by editing the [authentication DI module](../../internal/di/module/core/auth.go) (`provideAuthenticator`): replace the `default` fail-closed branch with `case config.EnvDevelopment / EnvStaging / EnvProduction` returning your real `Authenticator`.

### Authorization (authz)

This boilerplate ships an **allow-all** authorizer as a development stub. Implement a real policy decision point (PDP) for your project.

Create authorization functionality by implementing the usecase [Authorizer](../../internal/usecase/boundary/authz/authorizer.go) interface.

- Reference: [internal/infrastructure/authz/README.md](../../internal/infrastructure/authz/README.md)
- Stub example (allow-all): [internal/infrastructure/authz/allowall/authz_allowall.go](../../internal/infrastructure/authz/allowall/authz_allowall.go)
- Add your `stg` / `prd` implementations (RBAC from claims / ownership check / external policy engine such as OPA / Cedar) under `internal/infrastructure/authz/{stg,prd}/`.
- Wire them per environment by editing the [authorization DI module](../../internal/di/module/authz.go) (`provideAuthorizer`): replace the `default` fail-closed branch with `case config.EnvDevelopment / EnvStaging / EnvProduction` returning your real `Authorizer`.

The `Authorize(ctx, *auth.Authn, Action, *Resource)` signature already carries the full `Authn` (subject / scopes / claims) and the target `Resource` (with optional `OwnerID`), so both RBAC and ownership (object-level) models are expressible without changing call sites.

## Phase 12: Remove what only holds while this is a boilerplate

Two kinds of statement in this repository stop being true the moment you fork it: the passages
where it describes *itself* as a boilerplate, and the conventions it follows *because* it is one —
the in-place ADR amendment regime, the consolidation pass, the `setup-review` device. Both are
template scaffolding, not your project's documentation.

```sh
DRY_RUN=1 make setup-remove-boilerplate-identity
make setup-remove-boilerplate-identity
```

It scans the repository for `boilerplate-only` markers and resolves each one, deletes
[boilerplate-only-conventions.md](boilerplate-only-conventions.md) and its Japanese mirror, drops
its own make target from the registry, and then removes itself. It scans rather than working from
a list of files, because a list is something a marker can be written outside of — and a marker
nobody strips is a premise that survives into your project with nothing to announce it.

What it does **not** touch: the repository / module name (already replaced in Phase 5), and the
parts of this guide you keep reading — the clamped-config review and the exclusion ADRs below,
which several package READMEs link to.

What survives each removal is the general form of the rule, stated in the document that owns it:
[docs/adr/README.md](../adr/README.md), [docs/rules.md](../rules.md), the layer READMEs. Where a
statement needed a fork-appropriate replacement rather than plain deletion, the replacement is
already parked beside it and is swapped in by the same pass.

> Never strip `boilerplate-only` and `sample-api` markers in one run. They fire at different
> moments — this phase versus the sample removal in Phase 15 — and a fork may reasonably do one
> without the other.

## Phase 13: Review the template's deliberate exclusions (ADRs)

Beyond authentication / authorization (Phase 11) and deployment (Phase 10), this template makes other **deliberate non-choices** — for example: no in-application rate limiter, no generic cache abstraction, scheduled-job concurrency left to the scheduler, and push / streaming brokers kept out of the worker port.

Each such non-choice is recorded as an **exclusion ADR** under [docs/adr/](../../docs/adr/), tagged `setup-review`. List them with:

```sh
grep -rl "setup-review" docs/adr/
```

For your project, review each and decide:

- **Keep** — the exclusion fits your project; leave the ADR as is.
- **Change** — you need the opposite. Setup is where a fork establishes its **own baseline** from the template, so **edit the ADR directly** (rewrite its Decision / Consequences and update `deciders` / `date`) to record your project's choice, then implement accordingly.

The immutable, supersede-by-new-ADR model (do not edit; add a superseding ADR) applies to decisions you revisit **later**, during ongoing development — not to this one-time re-baselining at setup.

## Phase 14: Decide the dependency-license policy

The dependency-license scan (`make trivy-license`, and the `trivy-license` job in [.github/workflows/trivy-fs.yaml](../../.github/workflows/trivy-fs.yaml)) is **report-only, permanently**. It enumerates every dependency's license into the job summary and a PR comment, and never fails the build.

That is a deliberate non-choice, not an unfinished gate. Which licenses are acceptable is a legal question owned by the organization adopting this template: copyleft that is disqualifying for a distributed binary can be entirely acceptable for a service whose binary never leaves your infrastructure, and the answer varies by company, product, and distribution model. Picking a threshold here would bake one company's legal posture into every fork, so the template ships the inventory and leaves the judgement to the adopter.

If your organization has (or needs) a prohibited-license policy, gate it yourself:

1. Decide the acceptable set in terms of Trivy's own classification (`notice` / `unencumbered` / `permissive` / `reciprocal` / `restricted` / `forbidden` / `unknown`), and decide whether shipped artifacts and build-only tooling get the same bar. They may not: the classifications outside `notice` / `unencumbered` in this repository come from `docker/tools/`, which is build-only and never shipped.
2. Treat Trivy's classification as a starting point, not an authority. `BlueOak-1.0.0` lands in `unknown` even though it is OSI-approved and permissive, so decide such cases explicitly instead of letting the bucket decide for you.
3. Add the threshold to `trivy-license-ci` in [.makefiles/security/trivy.mk](../../.makefiles/security/trivy.mk) and a failing step to the `trivy-license` job, recording per-package exceptions in [.trivyignore.yaml](../../.trivyignore.yaml).
4. Update the trigger matrix in [.github/workflows/README.md](../../.github/workflows/README.md) and the license row of [ADR-0084](../adr/0084-multi-layer-security-scanning.md), which both currently state that no policy exists.

## Phase 15: Remove Sample APIs

This boilerplate includes sample APIs. Remove them according to your project requirements.

If you use AI-driven development, keeping sample APIs helps AI understand code structure and implementation patterns. You may also refactor sample APIs to align them with your project requirements.

### Removal Procedure

Use the automated command. It deletes the sample API (`user` / `product` / `order`) declared in [scripts/setup/remove-sample-api/sample-manifest.ts](../../scripts/setup/remove-sample-api/sample-manifest.ts), strips the `sample-api` marker blocks from the shared files (4 DI modules + `openapi.yaml`), and then regenerates / formats / lints.

> **The DB container must be running** before you run this — the final `gen-query` step dumps the **live** schema with `pg_dump`, so a stopped DB fails with `connection refused`.

```bash
# 0. Start the DB container (gen-query dumps the live schema)
docker compose up -d database

# Preview what will be removed (no changes are made)
DRY_RUN=1 make setup-remove-sample-api

# Remove the sample (deletes files + strips markers, then gen-api → gen-query → fix → lint)
make setup-remove-sample-api

# Rebuild the DB from the now sample-free migrations and re-dump the schema,
# so the dropped `users` table does not linger in models.gen.go
make db-init-local db-init-test
make gen-query
```

Notes:

- The base master data `prefecture` (migration `000001`, etc.) is **kept**.
- `gen-query` regenerates Go models from a `pg_dump` of the **live** DB. If you skip the DB rebuild above, the still-present `users` table is re-dumped and a stale `Users` type is regenerated into `models.gen.go` — the rebuild + re-`gen-query` is what actually drops it.
- Shared generated files (`*.gen.go`, `openapi.gen.yaml`, etc.) are not deleted directly — they are refreshed by the regeneration step.
- The sample is split into three domains: `user` is full-stack, while `product` / `order` currently exist only as DB stubs (migrations + product seeds). When you flesh `product` / `order` out into full APIs, append their new paths to the matching domain block in `sample-manifest.ts`, and wrap any sample lines interleaved in the shared files with `// sample-api:begin` … `// sample-api:end` (or a trailing `// sample-api:line`). They are then covered by the same command automatically.

### Rules keep their examples only until you remove the sample

Several rules in `docs/rules.md`, `docs/adr/**`, and the layer `README`s are stated in general terms
and then illustrated with a concrete example taken from the sample. **The rule survives removal; the
example does not.** What is left is a correct statement that no longer shows a reader what it looks
like in their own system.

Each of those places carries an HTML comment immediately above the removed line, stating **why an
example is needed there, what the example has to demonstrate, and how to write a replacement**. Find
them and work through the list:

```bash
grep -rn "撤去後にこの箇所へ自分の例を置くための指針" docs/ internal/ pkg/
```

This is not cosmetic. An abstract rule with no example is the form a rule takes right before people
stop applying it: every reader has to decide alone what it covers, and they decide differently. The
comments exist so that decision is made once, by you, with the original intent still in front of you.

The business vocabulary has its own home. The term table in
[`docs/spec/glossary.md`](../spec/glossary.md) is emptied by the same removal, and the rules for
filling it back in stay on the page.

<details>
<summary>Manual procedure (reference — no longer required)</summary>

1. Remove sample API definitions from [openapi.yaml](../../openapi/openapi.yaml)
    - Remove Path definitions under `サンプルAPI用のパス` and delete the referenced YAML files.
    - Remove Parameter definitions under `サンプルAPI用のパラメーター定義` and delete the referenced YAML files.
    - Remove Schema definitions under `サンプルAPI用の型定義` and recursively delete the referenced YAML files.
2. Remove sample API Controller and Usecase
    1. Run `make gen-api` to regenerate code and remove sample API Controller code.
    2. Delete Usecase files referenced by the sample API and their test files.
        - Also delete mock files.
    3. If there are files causing errors in [internal/integration](../../internal/integration/), delete those files as well.
    4. Delete handler files and test files affected by the absence of generated sample API code.
    5. If reference errors occur in the Infra layer (QueryService or CommandService interface errors), remove files used by the sample API and their test code from those interfaces.
3. Remove sample API Infra code
    1. Run `make db-test-migrate-down` and `make db-local-migrate-down` to reset the DB to a clean state.
    2. Delete execution SQL in `dml`.
        - Delete directories under [database/dml/repository](../../database/dml/repository).
        - Delete directories under [database/dml/query_service](../../database/dml/query_service).
        - Delete directories under [database/dml/command_service](../../database/dml/command_service).
    3. Run `make gen-query` to regenerate SQLC code and remove sample SQLC code.
    4. Remove sample Infra code and its test code that now cause errors.
4. Remove sample API domain code
    - Delete code used by the sample API and its test code under [internal/domain/](../../internal/domain/). Since directories under this path contain only sample domain code, you may delete entire directories.

</details>

## Phase 16: Decide your own ADR regime

Phase 12 removed the upstream's ADR conventions along with everything else that rested on this
being a boilerplate. What is left is a decision only you can take, because it is about how your
project records its own history rather than how this one shipped.

What you inherit is [docs/adr/README.md](../adr/README.md) as written: an ADR is an immutable
record, and a decision that changes is replaced by a new `accepted` ADR while the old one is marked
`superseded`. That is the ADR form as [MADR](https://adr.github.io/madr/) defines it, and what
[ADR-0000](../adr/0000-record-architecture-decisions.md) decided.

If you want in-place amendment instead — a legitimate choice for a design document that is shipped
rather than lived — record that as a decision of your own, in your own ADR. Do not infer it from
the fact that the upstream did it.
