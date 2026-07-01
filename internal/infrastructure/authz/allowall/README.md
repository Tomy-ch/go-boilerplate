# allowall (Allow-All Authorizer for Local Development)

English | [日本語](README.ja.md)

A trivial `Authorizer` implementation that **grants every request**. Intended for local development and CI / test environments only — **not for production use**.

## Role

- Satisfy the `Authorizer` dependency so feature development can proceed before a real authorization policy exists.
- `Authorize(...)` always returns `nil` (allow), regardless of subject / action / resource.

## Replacing for Production

The DI provider (`provideAuthorizer` in `internal/di/module/authz.go`) is environment-gated: it wires `allowall` only for local / CI / test, and returns an error for production-like environments so an allow-all policy can never ship. Replace it with an RBAC / external policy-engine (OPA / Cedar) implementation of `Authorizer`.

## Notes

- Performs no authorization at all (allows everything).
- Do not use in production.
