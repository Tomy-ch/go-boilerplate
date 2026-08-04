# httpheader

English | [日本語](README.ja.md)

Classification of HTTP header names.

## Public API

|Function|Description|
|---|---|
|`IsSensitive(name)`|Reports whether the header carries credentials and must not be forwarded outside the process.|

## Notes

`IsSensitive` matches on the header name with case and surrounding whitespace ignored, so a caller
that hands over `" Authorization"` or `"AUTHORIZATION"` is judged the same as `"authorization"`.
A single spelling would let the caller's formatting decide whether a credential leaves the process.

The set is deliberately small and fixed — `Authorization`, `Proxy-Authorization`, `Cookie`,
`Set-Cookie`. It answers "is this header a credential", not "is this value private": a header
carrying personal data is the caller's judgment and is not represented here.
