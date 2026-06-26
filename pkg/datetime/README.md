# datetime

English | [日本語](README.ja.md)

Provides date/time parsing utilities supporting multiple formats with timezone awareness.

## Role

Centralizes the layout strings and timezone handling involved in parsing date/time input, so callers across layers do not scatter raw `time.Parse` calls with inconsistent formats or location handling. This keeps date parsing a single, reusable, framework-agnostic concern with predictable behavior.

## Wraps

Standard library `time` package.
