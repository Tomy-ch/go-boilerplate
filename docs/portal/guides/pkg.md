# pkg

English | [日本語](README.ja.md)

`pkg/` is the directory for **shared utility packages** used across the entire application.

## Policy

`pkg/` only accepts new packages when they meet the following criteria:

- The functionality is **referenced from multiple locations**
- The package **wraps an external library** so that application code does not depend on it directly

Helpers used by only one feature should be placed within that feature's package.

Packages that perform external I/O (e.g. `exec`, `fs`, `xerrors`) follow a common
shape: define an **interface** for the capability, provide a concrete implementation
(`OS{}` / `stdErrors{}` etc.) that wires the real dependency, and add a
`//go:generate mockgen` directive so callers can inject a mock in tests.

### `pkg/` vs application-wide cross-cutting concerns

"Referenced from multiple locations" alone is **not** sufficient. `pkg/` is for
**context-independent, generic utilities** — code that could be lifted into any
project unchanged and carries no knowledge of this application's domain or system
decisions (e.g. `xerrors`, `uuid`, `ptr`, `stringkit`).

Concerns that are cross-cutting but **specific to this application/system** — the
application-wide error taxonomy (`internal/apperror`), logging (`internal/logging`),
observability (`internal/observability`), configuration (`internal/config`) — do
**not** belong in `pkg/` even though they are used across layers. They encode this
system's choices (error semantics, frameworks such as zap / otel) and therefore live
under `internal/` as cross-cutting concerns. The domain layer may depend on
`internal/apperror` as the one permitted such kernel.

### Constraints

- Must not contain business logic
- Must not depend on `internal/` packages
- Must not depend on infrastructure or framework-specific packages
- Each package must have a single responsibility

## Package List

|Package|Summary|Wraps|
|---|---|---|
|`datetime`|Date/time parsing|Standard library `time`|
|`envutil`|Environment variable override (test helper)|Standard library `os`|
|`exec`|External command execution (interface + mock)|Standard library `os/exec`|
|`fnmeta`|Function / package name extraction|Standard library `runtime`|
|`fs`|Filesystem operations (interface + mock)|Standard library `os`|
|`ptr`|Pointer operations|None|
|`safecast`|Type conversion with overflow detection|None|
|`stringkit`|String length validation|None|
|`uuid`|UUID value object|`github.com/google/uuid`|
|`xerrors`|Errors with stack traces|`github.com/cockroachdb/errors`|

## Package Details

### datetime

A parsing utility supporting multiple date/time formats.

Key functions

|Function|Description|
|---|---|
|`ParseRFC3339`|Parse RFC3339 format|
|`ParseRFC3339UTC`|Parse RFC3339 format (UTC)|
|`ParseRFC3339Nano`|Parse RFC3339Nano format|
|`ParseISO8601`|Parse ISO8601 format|
|`ParseDateTime`|Parse standard datetime format|
|`ParseDateOnly`|Parse date-only format|
|`ParseCustomLayout`|Parse with an arbitrary layout|

All functions have `ToLocation` variants (e.g. `ParseRFC3339ToLocation`) for parsing with a specified timezone.

### envutil

Temporarily overrides an environment variable and returns a restore function (mainly for tests / config loading).

|Function|Description|
|---|---|
|`Override(key, value)`|Set an env var and return a `func()` that restores the previous state|

### exec

Abstracts external command execution behind an interface so callers can inject a mock in tests. Production wires the `OS{}` implementation.

|Symbol|Description|
|---|---|
|`Runner` (interface)|`Output(ctx, dir, env, name, args)` — run a command and return stdout|
|`OS` (struct)|`os/exec`-based implementation of `Runner`|

### fnmeta

Decomposes full function names obtained from `runtime` to extract package and function names.

Primarily used for span name generation in `internal/observability`.

|Function|Description|
|---|---|
|`ExtractFunctionName`|Extract method name from full function name|
|`ExtractPackageName`|Extract package name from full function name|

### fs

Abstracts filesystem operations behind an interface so callers can inject a mock in tests. Production wires the `OS{}` implementation.

|Symbol|Description|
|---|---|
|`FS` (interface)|`ReadFile` / `WriteFile` / `Glob`|
|`OS` (struct)|`os`-based implementation of `FS`|

### ptr

Pointer manipulation utilities using generics.

|Function|Description|
|---|---|
|`To[T]`|Create a pointer from a value|
|`Copy[T]`|Copy a pointer (nil-safe)|
|`Deref[T]`|Dereference a pointer, returning a fallback when nil|

### safecast

Provides safe type conversion with overflow detection.

|Function|Description|
|---|---|
|`UintToInt`|Safe conversion from `uint` to `int`|

Returns `ErrOverflow` when an overflow occurs.

### stringkit

A set of validation functions based on string length (rune count).

|Function|Description|
|---|---|
|`RuneCount`|Return UTF-8 rune count|
|`InRange`|Check if length is within closed interval|
|`MaxOrLess`|Check if length <= max|
|`MinOrMore`|Check if length >= min|
|`StrictInRange`|Check if length is within open interval|
|`LessThanMax`|Check if length < max|
|`GreaterThanMin`|Check if length > min|

Each function has a corresponding `ErrorMsg` function for generating validation error messages.

### uuid

A UUID value object wrapping `github.com/google/uuid`.

Generates UUIDv7 and supports database integration (`sql.Scanner` / `driver.Valuer`).

|Function / Method|Description|
|---|---|
|`New`|Generate UUIDv7|
|`Parse`|Parse UUID from string|
|`NewTestFromSalt`|Generate deterministic UUID for testing|
|`String`|Return string representation|
|`IsNil`|Check if zero value|
|`Equal`|Compare UUIDs|
|`Scan` / `Value`|DB integration interface implementation|

### xerrors

Wraps `github.com/cockroachdb/errors` to provide error operations with stack traces.

|Function|Description|
|---|---|
|`New`|Create a new error|
|`Wrap`|Wrap an existing error|
|`Is`|Check error identity|
|`As`|Type-assert an error|
|`StackTrace`|Get stack trace string|

## Checklist for Adding a New Package

- [ ] Referenced from multiple locations, or wraps an external package
- [ ] Does not contain business logic
- [ ] Does not depend on `internal/`
- [ ] Single responsibility per package
- [ ] Tests are written
- [ ] Documentation is written
- [ ] Package summary is added to this README
