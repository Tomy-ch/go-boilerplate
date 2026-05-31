---
name: go-upgrade
description: Upgrade the Go version used by this project. Follows the procedure in `docs/maintenance/go-upgrade.md` and updates `.go-version`, `go.mod`, CI configs, Dockerfiles, dependencies, tooling, and generated code in order, then verifies with tests and lint. The target version is confirmed with the user at execution time.
---

# Go Version Upgrade Procedure

This skill defines the work procedure for upgrading the Go version used by this project to an arbitrary target version.

The canonical procedure document is:

- `docs/maintenance/go-upgrade.md`

A Japanese reference translation of this skill is available at `SKILL.ja.md` in the same directory (not loaded as a skill; for human reference only).

## First Step: Confirm the Target Version

This skill **MUST call `AskUserQuestion` immediately after invocation** to confirm the target version.
Even if a version-like string is present in the skill arguments or the most recent user message, do NOT adopt it silently and proceed (an explicit confirmation is required to prevent misconfiguration).

Procedure:

1. Read `.go-version` to determine the current version.
2. **Always** invoke `AskUserQuestion` to ask the user:
    - Question: "Specify the target Go version to upgrade to (e.g., `1.26.3`)."
    - Include the current version (the value of `.go-version`) as additional context.
    - If a version candidate is found in the skill arguments or recent message, include it in the question as "Candidate: `X.Y.Z`" and let the user confirm.
3. Validate that the answer is in `X.Y.Z` format. Use it as `<TARGET_VERSION>` throughout the rest of the procedure.

Do NOT modify any files or execute any commands until the target version has been confirmed.

## Preconditions

- Target version: `<TARGET_VERSION>` (the value confirmed above)
- Do NOT work directly on the `production` / `develop` / `staging` / `release/*` branches (see Git rules in AGENTS.md).
- Create a working branch from the latest `release/*` (e.g., `feature/go-upgrade-<TARGET_VERSION>`).

## AI Modification Scope

Per the "Exception: Skill Execution" clause in AGENTS.md, the normal AI Modification Scope restrictions are relaxed for the duration of this skill's execution. The following paths are permitted to be modified while this skill is running:

- `.go-version`
- `go.mod`, `go.sum`, `vendor/`
- `docker/**/Dockerfile` (and version strings in related README files)
- `.github/workflows/**` (only if a Go version is hard-coded there)

The following remain protected even during skill execution:

- `AGENTS.md` / `CLAUDE.md`
- Generated files (`**/*.gen.go`, `*.sql.go`, `*_mock.go`, `**/openapi.gen.yaml`, generated content under `docs/`)

## Execution Steps

Perform the following, mirroring the steps in `docs/maintenance/go-upgrade.md`. Replace every occurrence of `<TARGET_VERSION>` in commands and file contents with the version confirmed by the user.

### 1. Check the Release Notes

Check the release notes for `<TARGET_VERSION>` at <https://go.dev/doc/devel/release>.

Items to review:

- Language spec changes
- Breaking changes in the standard library
- Behavior changes in `go vet`
- Toolchain changes

### 2. Update `.go-version`

```text
<TARGET_VERSION>
```

### 3. Update the Local Go Environment

When using mise (recommended):

```sh
mise install
go version
```

When using goenv (alternative):

```sh
goenv install <TARGET_VERSION>
goenv local <TARGET_VERSION>
go version
```

When using Homebrew:

```sh
brew update
brew upgrade go
go version
```

The local environment update is a user task. The AI agent must NOT run these commands; instead, ask the user to perform this step.

### 4. Update the Go Version in CI

Inspect files under `.github/workflows`. If they use `go-version-file: go.mod`, no direct edit is required. If any workflow hard-codes the Go version, align it with `<TARGET_VERSION>`.

### 5. Update the Go Version in `go.mod`

```sh
go mod edit -go=<TARGET_VERSION>
```

### 6. Update Dependencies and Vendor

```sh
make tidy-lib
```

(Internally runs `go mod tidy` and `go mod vendor`.)

### 7. Reinstall Go Tools

```sh
make install-tools
```

### 8. Update the Dockerfile

Update any `golang:X.Y.Z` references under `docker/` to `<TARGET_VERSION>`. If the same version string appears in `docker/**/README.md`, update those as well.

### 9. Rebuild Docker Containers

Because the Go base image tag changes with this upgrade, use the clean (`--no-cache --pull`) variants so the new image is actually pulled and rebuilt:

```sh
make serve-build-clean
make tools-build-clean
```

### 10. Re-run Code Generation

```sh
make gen
```

### 11. Run Tests

```sh
make test
```

### 12. Run Lint

```sh
make lint
```

### 13. Final Verification

Confirm all of the following commands succeed:

```sh
make tidy-lib
make install-tools
make gen
make test
make lint
make serve-build-clean
make tools-build-clean
```

## Checklist

Confirm the following before reporting completion:

- [ ] Target version `<TARGET_VERSION>` confirmed with the user via `AskUserQuestion`
- [ ] Release notes reviewed
- [ ] `.go-version` updated to `<TARGET_VERSION>`
- [ ] Local Go updated (user task)
- [ ] CI Go version verified / updated
- [ ] `go.mod` Go version updated to `<TARGET_VERSION>`
- [ ] `make tidy-lib` executed
- [ ] `make install-tools` executed
- [ ] Dockerfile updated
- [ ] Docker containers rebuilt
- [ ] Code generation re-run
- [ ] Tests run
- [ ] Lint run

## Notes

- Do NOT manually edit generated code (`**/*.gen.go`, `*.sql.go`, `*_mock.go`, etc.).
- Commit on the working branch. Direct commits to `production` / `develop` / `staging` / `release/*` are prohibited.
- Push to a PR only when the user has explicitly instructed you to do so.
- After updating `SKILL.md`, also update `SKILL.ja.md` to keep the Japanese translation in sync.
