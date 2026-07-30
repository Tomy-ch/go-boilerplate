# core module

English | [日本語](README.ja.md)

`internal/di/module/core` provides **DI module groups for core components** commonly used in the HTTP stack.

Each module returns an `fx.Option` that registers the corresponding component in the DI container.

## Module List

|Function|File|Provided Component|
|---|---|---|
|`AuthnModule()`|`auth.go`|Authentication (Authenticator + Auth controller)|
|`BasicAuthModule()`|`basicauth.go`|Basic auth validator for metrics endpoint|
|`SecurityCookieModule()`|`security_cookie.go`|Cookie security attribute configuration|
|`SkipperModule()`|`skipper.go`|Skip OpenAPI validation for ops endpoints|
|`ValidatorModule()`|`validator.go`|OpenAPI schema validator|

## Design Policy

- Each module is isolated as one file = one module
- Internal implementation simply wraps constructors from `internal/controller/httpstack` etc. with `fx.Provide`
- Does not contain business logic

## Test Strategy

Every module here wraps pure constructors from `internal/controller/httpstack` and friends — nothing in
their closure needs real infrastructure to start. That affords a second tier of verification the parent
layer cannot afford, so each module carries two sibling tests:

- **`Test<Module>_GraphIsValid`** — the layer baseline declared in [`../README.md`](../README.md).
  `fx.ValidateApp` resolves the module together with its declared dependencies, proving the graph is
  wired with no missing types **without** executing constructors or lifecycle hooks.
- **`Test<Module>`** — starts a minimal `fx.New` app, `fx.Populate`s the provided component and asserts
  the value itself (that the environment gate selected `local.New()`, for instance). This reaches exactly
  what graph validation cannot: that the constructors actually run and yield a usable component.

The second tier is available **only because no module here needs real infrastructure to start**. That is
the criterion, not the directory: a module whose closure would require a database or a network connection
belongs in the parent, where `fx.ValidateApp` alone is the strategy.

Provider bodies carrying their own logic (`provideAuthenticator` / `provideJWKSAuthenticator`) are
unit-tested directly, per the DI layer baseline in [`../../README.md`](../../README.md) — graph validation
reaches neither, and the environment gate's refusal cases are the point.

## Notes

- Adding or removing modules requires updating references from the parent module in `internal/di/module`
