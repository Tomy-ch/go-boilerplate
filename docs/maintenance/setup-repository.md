# Post-Repository Clone Task List

English | [日本語](setup.ja.md)

For details of Make commands, refer to [Make Target List](.makefiles/README.md).

## Phase 1: Tool Setup

Install the tools required for VSCode development.

### 1.1. Install mise and activate it in your shell

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

### 1.2. Install the Go runtime and project tools

All tool versions (golangci-lint / sqlc / oapi-codegen / mockgen / dlv / lefthook / ...) are pinned in [`mise.toml`](../../mise.toml) as the single source of truth. The Dockerfiles, the local installer (`.makefiles/go/installer.mk`), and the CI workflows all install only what they need via `mise install <tool>` against the same `mise.toml`.

```sh
make go-update       # installs the pinned Go runtime
make install-tools   # installs gopls / gotests / impl / dlv / lefthook / golangci-lint
make activate-tools  # runs `lefthook install` to wire git hooks
```

## Phase 2: Local Startup Verification

Start the application locally and confirm it works without issues.

```sh
make serve
make tools
make db-init
```

## Phase 3: Execute Localization Script

Run the following commands to execute the script that replaces the Go module name in bulk.

Replace ORG and REPO as appropriate. Only change derived settings if necessary.

```sh
export ORG=<your-org/git-user-name>
export REPO=<your-repo>

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
make gen-api
make gen-sqlc
make tidy-lib
```

## Phase 4: Localization Verification

Confirm that basic functionality works correctly, including tests, static analysis, code generation, and health checks.

```sh
make test
make lint
make gen
curl http://localhost:8080/health
curl http://localhost:8080/ready
```

## Phase 5: Manual Rewrites

1. Rewrite the contents of README.md according to your project.
2. Rewrite the contents of [openapi.yaml](openapi/openapi.yaml) according to your project.
    - Rewrite the entire Info section according to your project.
        - title
        - termsOfService
        - contact
        - version
        - description
        - license

## Phase 6: Rewrite env Files

Rewrite the files in the [env/](env/) directory according to your project.

For the meaning of configuration values, refer to [env/README.md](env/README.md).

## Phase 7: Repository Initialization

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

## Phase 8: Create Deployment Configuration

This boilerplate adopts a configuration that does not depend on a specific cloud provider or deployment method, allowing flexible deployment to various cloud or on-premise environments.

Therefore, deployment settings do not include a specific deployment target. Add necessary settings according to your project's deployment destination.

Deployment CI/CD: Complete [.github/workflows/deploy-app.yaml](.github/workflows/deploy-app.yaml).

`Note: Please modify this section according to your environment` indicates sections that need to be modified according to your environment.

## Phase 9: Implement Authentication

This boilerplate includes sample code using JWT as an example implementation of authentication. Implement authentication according to your project requirements.

Create authentication functionality by implementing the usecase [Authenticator](internal/usecase/boundary/auth/authenticator.go) interface.

Refer to [internal/infrastructure/auth/README.md](internal/infrastructure/auth/README.md) for implementation.

Implementation example (local): [internal/infrastructure/auth/local/auth_local.go](internal/infrastructure/auth/local/auth_local.go)

After implementation, edit the [authentication DI module](internal/di/module/core/auth.go) to integrate authentication into the application.

## Phase 10: Remove Sample APIs

This boilerplate includes sample APIs. Remove them according to your project requirements.

If you use AI-driven development, keeping sample APIs helps AI understand code structure and implementation patterns. You may also refactor sample APIs to align them with your project requirements.

### Removal Procedure

Use the automated command. It deletes the sample API (`user` / `product` / `order`) declared in [scripts/setup/lib/sample-api.mjs](scripts/setup/lib/sample-api.mjs), strips the `sample-api` marker blocks from the shared files (4 DI modules + `openapi.yaml`), and then regenerates / formats / lints.

```bash
# Preview what will be removed (no changes are made)
DRY_RUN=1 make setup-remove-sample-api

# Remove, then regenerate / format / lint (runs make gen-api → gen-query → fix → lint)
make setup-remove-sample-api
```

Notes:

- The base master data `prefecture` (migration `000001`, etc.) is **kept**.
- Shared generated files (`*.gen.go`, `openapi.gen.yaml`, etc.) are not deleted directly — they are refreshed by the regeneration step.
- The sample is split into three domains: `user` is full-stack, while `product` / `order` currently exist only as DB stubs (migrations + product seeds). When you flesh `product` / `order` out into full APIs, append their new paths to the matching domain block in `sample-api.mjs`, and wrap any sample lines interleaved in the shared files with `// sample-api:begin` … `// sample-api:end` (or a trailing `// sample-api:line`). They are then covered by the same command automatically.

<details>
<summary>Manual procedure (reference — no longer required)</summary>

1. Remove sample API definitions from [openapi.yaml](openapi/openapi.yaml)
    - Remove Path definitions under `サンプルAPI用のパス` and delete the referenced YAML files.
    - Remove Parameter definitions under `サンプルAPI用のパラメーター定義` and delete the referenced YAML files.
    - Remove Schema definitions under `サンプルAPI用の型定義` and recursively delete the referenced YAML files.
2. Remove sample API Controller and Usecase
    1. Run `make gen-api` to regenerate code and remove sample API Controller code.
    2. Delete Usecase files referenced by the sample API and their test files.
        - Also delete mock files.
    3. If there are files causing errors in [internal/integration](internal/integration/), delete those files as well.
    4. Delete handler files and test files affected by the absence of generated sample API code.
    5. If reference errors occur in the Infra layer (QueryService or CommandService interface errors), remove files used by the sample API and their test code from those interfaces.
3. Remove sample API Infra code
    1. Run `make db-test-migrate-down` and `make db-local-migrate-down` to reset the DB to a clean state.
    2. Delete execution SQL in `dml`.
        - Delete directories under [database/dml/repository](database/dml/repository).
        - Delete directories under [database/dml/query_service](database/dml/query_service).
        - Delete directories under [database/dml/command_service](database/dml/command_service).
    3. Run `make gen-query` to regenerate SQLC code and remove sample SQLC code.
    4. Remove sample Infra code and its test code that now cause errors.
4. Remove sample API domain code
    - Delete code used by the sample API and its test code under [internal/domain/](internal/domain/). Since directories under this path contain only sample domain code, you may delete entire directories.

</details>
