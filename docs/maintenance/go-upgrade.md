# Go Version Upgrade Procedure

This document describes the **procedure for upgrading the Go version** used in this project.

Upgrading Go may affect several parts of the system, including:

- Go toolchain
- Dependency packages
- Go tools
- Code generation
- CI configuration
- Docker images

Therefore, follow the steps below when upgrading the Go version.

## 1. Check Release Notes

Before upgrading, review the official Go release notes for the target version.

Items to check:

- Changes to the language specification
- Breaking changes in the standard library
- Updates to `go vet`
- Toolchain changes

Example:

```text
https://go.dev/doc/devel/release
```

## 2. Update `.go-version`

This project manages the Go version using the `.go-version` file.

```text
.go-version
```

Example:

```text
1.26.0
```

## 3. Update Local Go Environment

This project **recommends using goenv** (but it is not required).

### Using goenv

```sh
goenv install 1.26.0
goenv local 1.26.0
```

Verify the installation:

```sh
go version
```

### Without goenv

If you use Homebrew:

```sh
brew update
brew upgrade go
```

Verify:

```sh
go version
```

## 4. Update Go Version in CI

Update the Go version used in GitHub Actions.

Target directory:

```text
.github/workflows
```

Example:

```yaml
- uses: actions/setup-go@v5
  with:
    go-version: '1.26.x'
```

## 5. Update Go Version in `go.mod`

Update the Go version declared in `go.mod`.

```sh
go mod edit -go=1.26
```

## 6. Update Dependencies and Vendor

This project uses the **Makefile task `tidy-lib`** to manage dependencies.

```sh
make tidy-lib
```

This task runs:

- `go mod tidy`
- `go mod vendor`

## 7. Reinstall Go Tools

When upgrading Go, previously installed tools remain built with the older Go version.

Reinstall them to ensure compatibility.

```sh
make install-tools
```

Tools typically installed include:

- gopls
- golangci-lint
- delve
- lefthook
- gotests
- impl
- goplay

## 8. Update Docker Image

Update the Go version used in the Dockerfile.

Example:

```dockerfile
FROM golang:1.26
```

## 9. Rebuild Docker Containers

This project provides Makefile tasks to rebuild Docker images.

Server containers:

```sh
make serve-build
```

Tool containers:

```sh
make tools-rebuild
```

## 10. Regenerate Code

Changes in the Go version may affect generated code.

Run code generation again.

```sh
make gen
```

## 11. Run Tests

Execute the test suite.

```sh
make test
```

or

```sh
go test ./...
```

## 12. Run Lint

Run lint checks to ensure code quality.

```sh
make lint
```

## 13. Final Verification

Ensure all the following commands succeed.

```sh
make tidy-lib
make install-tools
make gen
make test
make lint
make serve-build
make tools-rebuild
```

## Upgrade Checklist

When upgrading the Go version, verify the following:

- [ ] Review Go release notes
- [ ] Update `.go-version`
- [ ] Update local Go environment
- [ ] Update Go version in CI
- [ ] Update Go version in `go.mod`
- [ ] Run `make tidy-lib`
- [ ] Run `make install-tools`
- [ ] Update Dockerfile
- [ ] Rebuild Docker containers
- [ ] Regenerate code
- [ ] Run tests
- [ ] Run lint
