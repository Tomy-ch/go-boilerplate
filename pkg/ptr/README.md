# ptr

English | [日本語](README.ja.md)

Pointer manipulation utilities using generics.

## Public API

|Function|Description|
|---|---|
|`To[T](v T) *T`|Create a pointer from a value|
|`Copy[T](v *T) *T`|Shallow-copy the pointed-to value into a new pointer (returns nil if input is nil; reference-type fields are shared)|
|`Deref[T](p *T, fallback T) T`|Dereference a pointer, or return fallback if nil|

## Notes

Requires Go 1.18+ (generics).
