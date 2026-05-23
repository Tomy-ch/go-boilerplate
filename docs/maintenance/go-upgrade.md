# Go Version Upgrade Procedure

This document explains the **procedure for updating the Go version** in this project.

Updating the Go version affects the following.

- Go toolchain
- Dependency packages
- Go tools
- Code generation
- CI
- Docker images

Therefore, update according to the following steps.

## 1. Check Release Notes

Go programming language

Check the Release Notes of the target version.

Main items to check

- Changes to the language spec
- Breaking changes in the standard library
- Changes to `go vet`
- Changes to the toolchain

Example

```text
<https://go.dev/doc/devel/release>
```

## 2. Update `.go-version`

This project manages the Go version using `.go-version`.

```text
.go-version
```

Example

```text
1.26.1
```

## 3. Update Local Go Environment

This project **recommends using goenv (not mandatory)**.

### When using goenv

```sh
goenv install 1.26.1
goenv local 1.26.1
```

Verification

```sh
go version
```

### When not using goenv

If using Homebrew

```sh
brew update
brew upgrade go
```

Verification

```sh
go version
```

## 4. Update Go version in CI

Update the Go version in GitHub Actions.

Target directory

```text
.github/workflows
```

Example

```yaml

- uses: actions/setup-go@v6
  with:
    go-version-file: go.mod
    cache: true
```

## 5. Update Go version in `go.mod`

```sh
go mod edit -go=1.26.1
```

## 6. Update dependencies and vendor

This project uses **Makefile task `tidy-lib`** for dependency management.

```sh
make tidy-lib
```

This task executes the following.

- `go mod tidy`
- `go mod vendor`

## 7. Reinstall Go tools

When the Go version is updated, Go tools remain as binaries built with the old Go version.

Therefore, reinstall the tools.

```sh
make install-tools
```

Main tools installed

- gopls
- golangci-lint
- delve
- lefthook
- gotests
- impl
- goplay

## 8. Update Docker image

Update the Go version in the Dockerfile.

Example

```dockerfile
FROM golang:1.26.1
```

## 9. Rebuild Docker containers

Because the Go base image tag changes with this upgrade, use the clean (`--no-cache --pull`) variants so the new image is actually pulled and rebuilt.

Server containers

```sh
make serve-build-clean
```

Tool containers

```sh
make tools-build-clean
```

## 10. Re-run code generation

Generated code may change due to Go version changes.

```sh
make gen
```

## 11. Run tests

```sh
make test
```

or

```sh
go test ./...
```

## 12. Run lint

```sh
make lint
```

## 13. Final check

Ensure that all of the following commands succeed.

```sh
make tidy-lib
make install-tools
make gen
make test
make lint
make serve-build-clean
make tools-build-clean
```

## Upgrade Checklist

When updating the Go version, check the following.

- [ ] Check Release Notes
- [ ] Update `.go-version`
- [ ] Update local Go
- [ ] Update CI Go version
- [ ] Update `go.mod` Go version
- [ ] Run `make tidy-lib`
- [ ] Run `make install-tools`
- [ ] Update Dockerfile
- [ ] Rebuild Docker containers
- [ ] Re-run code generation
- [ ] Run test
- [ ] Run lint
