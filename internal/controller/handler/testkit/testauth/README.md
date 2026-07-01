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
