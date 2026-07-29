# patch

English | [日本語](README.ja.md)

Three-state values for partial-update (PATCH) input.

## Role

A partial update needs to distinguish three cases per field: the field was not sent (keep the current value), it was sent as `null` (clear the value), and it was sent with a value (replace it). A plain `*T` collapses the first two into `nil`, so a request that clears a field becomes indistinguishable from one that omits it.

`Field[T]` carries that distinction across the boundary between outer-layer DTOs and domain values, and `Resolve` applies it to a current value so callers do not re-implement the three-way branch per field.

## Public API

- `Unspecified[T]() Field[T]` — the field was not sent.
- `Null[T]() Field[T]` — the field was sent as `null`.
- `Value[T](v T) Field[T]` — the field was sent with a value.
- `(Field[T]) Resolve(current *T) *T` — the value to persist, given the current one.

## Notes

The zero value of `Field[T]` is unspecified, so a struct of `Field` values defaults to "change nothing".

This package models only the shape of the input. Deciding which fields a given API accepts as clearable, and validating the resolved values, belong to the layers that own those rules.

Requires Go 1.18+ (generics).
