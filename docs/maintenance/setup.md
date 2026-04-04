# Post-Repository Clone Task List

English | [日本語](setup.ja.md)

For details of Make commands, refer to [Make Target List](.makefiles/README.md).

## Phase 1: Tool Setup

Install the tools required for VSCode development.

The versions are listed in `tools.yaml` as the versions that have been verified to work. Change these versions as necessary.

```sh
make sync-tools
make install-tools
make activate-tools
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

## Phase 9: Implement Authentication

This boilerplate includes sample code using JWT as an example implementation of authentication. Implement authentication according to your project requirements.

Create authentication functionality by implementing the usecase [Authenticator](internal/usecase/boundary/auth/authenticator.go) interface.

Refer to [internal/infrastructure/auth/README.md](internal/infrastructure/auth/README.md) for implementation.

Implementation example (local): [internal/infrastructure/auth/local/auth_local.go](internal/infrastructure/auth/local/auth_local.go)

After implementation, edit the [authentication DI module](internal/di/module/core/auth.go) to integrate authentication into the application.

## Phase 10: Remove Authentication Debug APIs

Authentication debug APIs pose security risks, so remove them as necessary.

It is not mandatory to remove them during setup, but you MUST remove them before releasing to production.

```sh
make setup-remove-debug-handlers
```

## Phase 11: Remove Sample APIs

This boilerplate includes sample APIs. Remove them according to your project requirements.

If you use AI-driven development, keeping sample APIs helps AI understand code structure and implementation patterns. You may also refactor sample APIs to align them with your project requirements.

### Removal Procedure

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

## Phase Extra: About IP Rate Limiting

In this project, an `in-memory IP rate limiter` is provided as a sample implementation.

However, this approach is not suitable for environments with multiple instances or frequent instance restarts.

If unnecessary, delete the following:

- Remove the line `security.RateLimitModule(),` from RateLimitModule in [internal/di/server/server.go](../../internal/di/server/server.go)
- Delete the entire file [internal/di/server/extension/security/rate_limit_di.go](../../internal/di/server/extension/security/rate_limit_di.go)
- Delete the entire file [internal/di/server/extension/security/rate_limit_di_test.go](../../internal/di/server/extension/security/rate_limit_di_test.go)
- Delete the entire directory [internal/controller/httpstack/ratelimit/](../../internal/controller/httpstack/ratelimit/)
