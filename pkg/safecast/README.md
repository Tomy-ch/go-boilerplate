# safecast

English | [日本語](README.ja.md)

Provides safe type conversion with overflow detection.

## Role

Go's built-in numeric conversions silently wrap or truncate on overflow. This package centralizes range-checked conversions so that crossing signed/unsigned or width boundaries fails loudly with an error instead of corrupting a value, giving every layer one trustworthy, framework-agnostic conversion to reach for.

## Notes

Returns `ErrOverflow` when the value exceeds `math.MaxInt`.
