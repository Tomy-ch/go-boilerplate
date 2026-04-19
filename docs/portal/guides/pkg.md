# pkg

English | [日本語](README.ja.md)

`pkg/` is the directory for **shared utility packages** used across the entire application.

## Policy

`pkg/` only accepts new packages when they meet the following criteria:

- The functionality is **referenced from multiple locations**
- The package **wraps an external library** so that application code does not depend on it directly

Helpers used by only one feature should be placed within that feature's package.

### Constraints

- Must not contain business logic
- Must not depend on `internal/` packages
- Must not depend on infrastructure or framework-specific packages
- Each package must have a single responsibility

## Package List

|Package|Summary|Wraps|
|---|---|---|
|`datetime`|Date/time parsing|Standard library `time`|
|`fnmeta`|Function / package name extraction|Standard library `runtime`|
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

All functions have `InLocation` variants for parsing with a specified timezone.

### fnmeta

Decomposes full function names obtained from `runtime` to extract package and function names.

Primarily used for span name generation in `internal/observability`.

|Function|Description|
|---|---|
|`ExtractFunctionName`|Extract method name from full function name|
|`ExtractPackageName`|Extract package name from full function name|

### ptr

Pointer manipulation utilities using generics.

|Function|Description|
|---|---|
|`To[T]`|Create a pointer from a value|
|`Copy[T]`|Copy a pointer (nil-safe)|

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
