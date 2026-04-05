# security Package

Overview: This package provides security-related functionality, including encryption.

It implements `BcryptHasher` to perform password hashing and comparison.

## Main Features Provided

- `NewBcryptHasher()` function: Generates a `BcryptHasher` for hashing passwords.
- `Hash(password string) (string, error)` method: Hashes a password.
- `Compare(hash, password string) (bool, error)` method: Compares a hashed password with a plaintext password.

## Usage

Implement an appropriate `BcryptHasher` for each environment or service to ensure application security.

To integrate into the system, add the implementation to `security` in `internal/di/module/infrastructure.go`.
