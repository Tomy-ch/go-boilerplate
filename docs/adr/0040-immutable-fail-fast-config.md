---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [config]
---

# ADR-0040: Config is immutable, loaded once at startup, fail-fast

## Status

accepted

## Context

Configuration that can be read at any time or mutated after startup creates race
conditions, obscures what value was in effect during a failure, and makes components
harder to test because their configuration context is unpredictable.

Additionally, a service that starts with invalid or missing configuration can run for
minutes before a missing credential or wrong host causes a failure deep in a request
path. Detecting problems early — before any traffic is served — produces clearer error
messages and prevents partial initialization.

## Decision

Configuration is loaded **exactly once** at application startup through `config.SetUpConfig()`,
which chains `config.Load()` (reads the embedded `env/.env` into process env vars) then
`config.New()` (parse + validate). The loading sequence is:

```text
config.Load(): env/.env (embedded) → process env
config.New():  env.ParseAs[Loader]() → validateConfig() → *Config
```

The returned `*Config` type is **immutable**: it exposes only getter methods; no setter
methods exist. Components receive narrowly scoped SubConfig values (e.g.,
`*config.ServerConfig`, `*config.DatabaseConfig`) via DI providers rather than the full
`*Config` object.

`validateConfig()` enforces constraints (port ranges, CIDR format, timeout ordering,
allowed-origins non-empty, etc.) and returns an error immediately if any constraint is violated.
The application **fails to start** on any validation error or missing required field,
rather than running in a degraded or misconfigured state.

Domain and usecase layers must not depend on environment variables directly; configuration
interpretation is confined to the `config` package.

## Consequences

### Positive Consequences

- The effective configuration at any point in time is fixed and observable from startup
  logs; no runtime mutation can change the values a component sees.
- Invalid or incomplete configuration surfaces immediately at startup with a clear error,
  not mid-request.
- SubConfig providers expose only the fields a component needs, reducing its visible
  surface and making test setup straightforward.

### Negative Consequences

- Changes to configuration require a process restart; there is no hot-reload mechanism.
- Test code must use the provided testing helpers (`MockConfigForTest`, `t.Setenv`) to
  inject different values; direct field mutation is intentionally unavailable.

## Alternatives Considered

### Mutable config with live reload

Allow configuration to be reloaded at runtime (e.g., from a file watcher or a config
server). Rejected: increases complexity, introduces race conditions, and requires every
consumer to handle value changes at arbitrary times.

### Load config on first use (lazy)

Defer loading until a component first reads a value. Rejected: defers detection of
missing or invalid values until that component is exercised, which may not happen during
startup verification.

## Notes

- Source: `internal/config/README.md` (Design Principles section),
  `internal/config/setup.go` (`SetUpConfig()`), `internal/config/loader.go` (`Load()`),
  `internal/config/config.go` (`New()` and `validateConfig()`).
- Subsystem struct layout: [ADR-0038](0038-subsystem-typed-config-loaders.md).
- Default-vs-required governance: [ADR-0039](0039-config-default-vs-required-governance.md).
- Embedded env file mechanism: [ADR-0041](0041-embedded-self-contained-binary.md).
