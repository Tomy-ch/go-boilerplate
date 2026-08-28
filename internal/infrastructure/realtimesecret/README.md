# realtimesecret

Implements `boundary/realtime.SecretGenerator` — the opaque ticket value of Realtime Delivery — from
the operating system's cryptographically secure randomness source: 32 random bytes rendered as
base64url without padding (43 characters). The value carries no structure and no claims; the store
keeps only its hash, so it can be checked only by presenting it.

It duplicates the shape of [`token`](../token/README.md) on purpose rather than depending on it:
`boundary/token` belongs to the cart's session tracking and is removed with the sample feature, and
Realtime Delivery must keep compiling and testing after that removal.

## Notes

- No tracer span — no real I/O happens here (see [Observability](../README.md)).
- `crypto/rand`, never `math/rand`; a predictable ticket is not a credential.
- A short read is an error, not a shorter value.

## Test strategy

Unit tests only: the encoding and width are asserted on real output, uniqueness over repeated calls,
and the error path through an injected failing reader. There is no substrate to contract-test.
