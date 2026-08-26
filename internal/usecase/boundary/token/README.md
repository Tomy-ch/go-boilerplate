# token

English | [日本語](README.ja.md)

Provides a `Generator` interface for producing unguessable, opaque token strings.

```go
type Generator interface {
    Generate() (string, error)
}
```

## Why Abstract?

Randomness is an effect, so a caller that reaches for it directly stops being reproducible: the
same inputs no longer produce the same result, and a test can only assert shape rather than value.
This is the same reason `Clock` exists for time — see
[Time Handling Policy](../../README.md#time-handling-policy) for the argument in full.

Abstracting it also keeps the two halves of a token in their own layers. How long a token is and
which alphabet it uses are properties of the value being generated, so they stay with the
implementation; whether a given string is an acceptable token is a rule about the domain's own
value object, which validates what it is handed. Neither half needs to know the other, and the
generator returns a plain string rather than a domain type so the boundary carries no business
vocabulary.

`Generate` takes no argument. A caller that needs a token needs *a* token, and no caller yet asks
for a particular length — the same reason `Clock.Now()` takes none. It returns an error because
reading from the system's randomness source can genuinely fail, and a failure that is not
logically unreachable is propagated rather than turned into a panic.

## Implementation

`internal/infrastructure/token/` provides the concrete implementation, which reads from the
operating system's cryptographically secure randomness source.
