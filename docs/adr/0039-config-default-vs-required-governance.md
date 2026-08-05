---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [config, governance]
---

# ADR-0039: Governance: default-in-code (immutable) vs required-in-file (variable)

## Status

accepted

## Context

Config fields in `internal/config/envspec.go` are backed by environment variables. For
each field, a decision must be made about whether the value is:

- **a universal framework constant** — a sensible default that every project derived from
  the boilerplate keeps unchanged (e.g., a database driver name, a timeout that works
  for most workloads), or
- **a project-specific or per-environment value** — something that differs between
  projects, environments, or deployment targets and must therefore be set explicitly
  (e.g., database host, allowed CORS origins, authentication credentials).

Without a documented rule, contributors make inconsistent choices: some values that
should be required are given defaults and silently pick up wrong values in production;
some values that should be fixed constants are moved to env files, creating unnecessary
operator burden.

## Decision

Two distinct categories govern how a config value is supplied:

**Code default (immutable):** Fields carrying a `envDefault` tag in `envspec.go` are
intentionally omitted from `.env` files. They are framework-level constants that any
project derived from this boilerplate is expected to keep unchanged. The default applies
automatically; an explicit `.env` entry is added only when a project genuinely needs to
override it. These are marked **Code default `<value>`** in `env/README.md`.

**Required in file (variable):** Fields marked `required` in `envspec.go` have no
embedded default. Every such variable must be present in `env/.env` (the local default)
and in every per-environment file (`env/.env.<env>`). Missing a required field causes
`env.ParseAs` to fail, aborting startup (consistent with the fail-fast principle in
[ADR-0040](0040-immutable-fail-fast-config.md)).

The rule for choosing the category when adding a new variable (from `env/README.md`):

- **Project-specific or per-environment value** → mark `required`; add to `env/.env` and
  every per-environment file.
- **Universal framework default** → use `envDefault`; omit from `.env` files; mark as
  **Code default** in the table.

Examples from `envspec.go`:

```go
// required: differs per project / environment
Env  string `env:"ENV,required"`
Host string `env:"HOST,required"`

// code default: universal constant, rarely overridden
ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"45s"`
Driver          string        `env:"DRIVER"           envDefault:"pgx"`
```

## Consequences

### Positive Consequences

- The intent of each config field is explicit and machine-checkable: `required` prevents
  startup with a missing value; `envDefault` prevents `.env` clutter for stable
  constants.
- Operators know from the `env/README.md` table exactly which variables they must set
  and which they can ignore.

### Negative Consequences

- The category choice is a judgment call for each field; borderline cases (a value that
  most projects keep the same but some override) require deliberate classification.
- Changing a field from `envDefault` to `required` (or vice versa) is a breaking change
  for any project that relied on the old behavior.

## Alternatives Considered

### All values required

Explicit for every deployment, but drowns operators in variables that never change
across projects. Rejected.

### All values have defaults

No required fields; the application can always start. Rejected: production misconfiguration
(wrong host, missing credentials) would produce a running but broken service rather than
a startup failure.

## Notes

- Source: `env/README.md` (Conventions section and "Adding a New Variable" step 3),
  `internal/config/envspec.go` (field-level tag examples).
- Fail-fast startup behavior: [ADR-0040](0040-immutable-fail-fast-config.md).
- Subsystem struct layout: [ADR-0038](0038-subsystem-typed-config-loaders.md).
