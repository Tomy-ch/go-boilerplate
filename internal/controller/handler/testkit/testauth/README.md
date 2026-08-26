# testauth

English | [日本語](README.ja.md)

Provides utilities for setting authentication context inside test code that is normally populated by middleware.

## Usage

Use the `MakeAvailableAuthn` function to attach authentication context to a test context.

```go
  ctx := context.Background()
  ctx = testauth.MakeAvailableAuthn(ctx, t, userID.String()) // set auth context here
  ctrl := gomock.NewController(t)
```

This function attaches an authentication context carrying the given user ID, making it available to controllers under test.

The subject doubles as the internal UserID: when it parses as a UUID the returned `Authn` has its UserID resolved, and otherwise it stays unresolved — which is how a test reaches the "authenticated but no internal user" path. A zero-value UUID subject fails the test, because `auth.Authn` refuses to resolve one (`ErrUserIDZero`).
