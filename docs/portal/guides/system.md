# internal/system

English | [日本語](README_ja.md)

`internal/system` provides **runtime metadata (build information)** for the application.

This package handles **process metadata** that is independent of business logic and infrastructure.  
It allows the application to access **version information, Git revision, and build timestamp**.

These values are typically **injected at build time using Go `ldflags`**.

## Responsibility

The responsibilities of this package are:

- Store **application build metadata**
- Provide runtime access to **Version / Revision / BuildDate**
- Provide APIs usable by version display (`--version`) or diagnostic endpoints
- Allow **BuildInfo to be mocked in tests**

This package **does not contain business logic**.

## Package Structure

```txt
internal/system
├── buildinfo.go
├── version.go
└── mock/
```

|File|Role|
|---|---|
|`buildinfo.go`|BuildInfo interface and implementation|
|`version.go`|Build metadata embedded at build time|
|`mock/`|Mocks used for testing|

## BuildInfo Interface

Application code retrieves build information through the **BuildInfo interface**.

```go
type BuildInfo interface {
 Version() string
 Revision() string
 BuildDate() string
}
```

Implementation:

```go
func NewBuildInfo() BuildInfo
```

Usage example:

```go
bi := system.NewBuildInfo()

version := bi.Version()
revision := bi.Revision()
buildDate := bi.BuildDate()
```

This design enables:

- Mock injection during testing
- Abstraction of version retrieval
- Dependency injection through DI

## version.go

`version.go` defines variables that are **overwritten at build time**.

```go
var (
 Version   = "dev"
 Revision  = "none"
 BuildDate = "2024-12-31T21:00:00Z"
)
```

These default values serve as **fallback values for development environments**.

Actual values are overridden during `go build`.

## Build-Time Value Injection

CI / Docker / Makefile typically inject values as follows.

Example:

```bash
go build \
  -ldflags "-X 'internal/system.Version=1.2.3' \
            -X 'internal/system.Revision=abcdef1' \
            -X 'internal/system.BuildDate=2025-01-01T00:00:00Z'"
```

Example in Dockerfile:

```bash
ARG VERSION
ARG REVISION
ARG BUILD_DATE

RUN go build \
  -ldflags "-X 'internal/system.Version=$VERSION' \
            -X 'internal/system.Revision=$REVISION' \
            -X 'internal/system.BuildDate=$BUILD_DATE'"
```

## Usage

BuildInfo is used for the following purposes.

Examples:

```txt
--version command
/version API
/health endpoint
log output
diagnostic information
```

Example output:

```txt
service version=1.2.3 revision=abc123 build=2025-01-01
```

## Layer Position

`internal/system` belongs to the **runtime metadata layer of the application**.

```txt
Controller
Usecase
Domain
Infrastructure
System (runtime metadata)
```

Characteristics:

- Contains no business logic
- Does not depend on Infrastructure
- Accessible from the entire application

## Testing

Because `BuildInfo` is defined as an interface, tests can use mocks.

```bash
go:generate mockgen
```

Example:

```go
mockBuildInfo := mock_system.NewMockBuildInfo(ctrl)
mockBuildInfo.EXPECT().Version().Return("1.0.0")
```

## Security Considerations

Build metadata must **not include** the following:

- authentication tokens
- environment variables
- private keys
- personal information

Recommended metadata to include:

- Version
- Git Revision
- Build Date

## Summary

`internal/system` provides:

- build metadata
- version management
- runtime metadata retrieval
- testable abstractions

It functions as the **runtime metadata package for the application**.
