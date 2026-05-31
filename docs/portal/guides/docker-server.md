# server Dockerfile

English | [日本語](README.ja.md)

This Dockerfile defines the application server images. It uses multi-stage builds to provide targets for production, migration, and local development.

## Role

`docker/server/Dockerfile` is the single source of truth for every server-side container the project ships. One Dockerfile produces four targets (`builder` / `runtime` / `migration` / `tooling`) so that production deploys, schema migration jobs, and local hot-reload development all stay aligned on the same base layers and Go toolchain version. Keeping these in one place avoids drift between "what runs in production" and "what runs on a developer's laptop" while letting each target add only what it needs (e.g., dev tools live only in `tooling`).

## Build Targets

|Target|Base Image|Purpose|
|---|---|---|
|`builder`|`golang:1.26.3-alpine`|Build the Go binary with `ldflags` (version/revision/build date)|
|`runtime`|`alpine:3.23`|Production runtime container (non-root `app` user)|
|`migration`|Inherits `runtime`|Migration execution container (`migrate-up` command)|
|`tooling`|`golang:1.26.3-alpine`|Local development environment (hot reload + debugging)|

## runtime

- Runs as non-root user (`app`)
- Builds with `vendor` mode (`GOPROXY=off`)
- Embeds version / revision / build date via `-ldflags`
- Default command: `./server serve`

## migration

- Inherits `runtime` image
- Adds `database/migrations` directory
- Default command: `./server migrate-up`
- Intended as a one-shot job before application deployment

## tooling (Local Development)

Pre-installed tools:

|Tool|Purpose|
|---|---|
|`air`|Hot reload|
|`dlv`|Go debugger|
|`golines`|Line-length-limited formatting|
|`gofumpt`|Enhanced gofmt|
|`golangci-lint`|Go linter|

OS-level packages: `build-base`, `binutils-gold`, `bash`, `curl`, `git`, `upx`, `libc6-compat`, `gcompat`, `tzdata`, `make`

Default command: `air -c .air.toml`

## docker-compose Service

```yaml
api_server:
  dockerfile: docker/server/Dockerfile
  target: tooling
  ports: 8080 (API), 2345 (dlv), 6060 (pprof)
```

## Notes

- Production images should pin base image digests for reproducibility
- `tooling` target uses `@latest` for tools during initial development — pin versions for CI parity
- Working directory is `/app` for all targets
