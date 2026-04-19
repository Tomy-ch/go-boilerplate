# ptr

English | [日本語](README.ja.md)

Pointer manipulation utilities using generics.

## Public API

|Function|Description|
|---|---|
|`To[T](v T) *T`|Create a pointer from a value|
|`Copy[T](v *T) *T`|Copy a pointer (returns nil if input is nil)|

## Notes

Requires Go 1.18+ (generics).
