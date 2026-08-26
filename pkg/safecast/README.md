# safecast

English | [日本語](README.ja.md)

Provides safe type conversion with overflow detection.

## Role

Go's built-in numeric conversions silently wrap or truncate on overflow. This package centralizes range-checked conversions so that crossing signed/unsigned or width boundaries fails loudly with an error instead of corrupting a value, giving every layer one trustworthy, framework-agnostic conversion to reach for.

## Notes

Returns `ErrOverflow` when the value falls outside the destination type's range.

Optional values have a pointer form (`IntPtrToInt32Ptr`), so that "absent stays absent" is centralized here too instead of being re-implemented at every nullable boundary. `nil` passes through as `nil`, and a non-nil result addresses a fresh value — mutating the argument afterwards cannot alter it.

Depends on `pkg/xerrors` for error wrapping — the sole permitted `pkg/` → `pkg/` dependency (enforced by depguard `independent_pkg`).
