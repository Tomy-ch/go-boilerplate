# security

English | [日本語](README.ja.md)

Provides a `Hasher` interface for password hashing and comparison.

```go
type Hasher interface {
    Hash(password string) (string, error)
    Compare(hash, password string) (bool, error)
}
```

## Design Intent

- Hide cryptographic algorithm details (bcrypt, argon2, etc.) from Usecase
- Enable algorithm replacement without affecting business logic
- Allow mock substitution in tests

## Implementation

`internal/infrastructure/security/` provides a bcrypt-based implementation.

## Notes

- `Compare` returns `(false, nil)` for mismatched passwords — mismatch is not an error
- bcrypt cost is configured via `config.SecurityConfig.BcryptCost()`
