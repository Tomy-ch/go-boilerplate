# ptr

English | [日本語](README.ja.md)

Pointer manipulation utilities using generics.

## Role

Centralizes nil-safe conversions between values and pointers and defensive copying of pointer fields, so callers do not repeat manual nil-checks or accidentally share mutable pointer state. These conversions recur wherever optional fields cross the boundary between outer-layer DTOs and domain values, so a single generic helper keeps that repetitive conversion code consistent and framework-agnostic.

## Notes

Requires Go 1.18+ (generics).
