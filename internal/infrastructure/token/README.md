# token

English | [日本語](README.ja.md)

Implements the `boundary/token.Generator` interface using the operating system's
cryptographically secure randomness source.

## Responsibility

Produces an opaque token string: 32 random bytes rendered as base64url without padding, which
yields 43 characters. Nothing else — the value carries no structure, no encoded claims, and no
expiry, so it can only be checked by looking it up.

Both properties of the value live here. 256 bits is the width at which guessing a token is not a
realistic attack, and base64url is the encoding that survives being placed in a URL or a header
unescaped. The domain's `cart.SessionToken` independently validates what it is handed — the length
and the alphabet — so a change on either side that breaks the agreement fails a test rather than
producing tokens the domain silently rejects.

## Notes

- No tracer span. This component performs no real I/O, so an infrastructure span would record
  nothing but its own overhead — see [Observability](../README.md) for where that line is drawn.
- `crypto/rand` is used rather than `math/rand`. A token that an attacker can predict is not a
  token, and only the cryptographic source makes that claim.
- A read failure is returned, not swallowed. It is rare but genuinely possible, and the caller is
  the one that knows whether it can proceed without a token.
