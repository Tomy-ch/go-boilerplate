# allowall (Allow-All Authorizer for Local Development)

English | [日本語](README.ja.md)

A trivial `Authorizer` implementation that **grants every request**. Intended for local development and CI / test environments only — **not for production use**.

## Role

- Satisfy the `Authorizer` dependency so feature development can proceed before a real authorization policy exists.
- `Authorize(...)` always returns `nil` (allow), regardless of subject / action / resource.

## Fail-closed Construction

`New` takes the `*config.ApplicationConfig` and **refuses to construct outside `local` / `ci` / `test`**, returning an error for production-like environments. Because a stub that grants everything is dangerous, this precondition is owned by the stub itself rather than by its callers — a wiring mistake in `provideAuthorizer` cannot make an allow-all policy reachable in production. The DI provider surfaces the refusal as a startup failure.

## Replacing for Production

Replace `allowall` with an RBAC / external policy-engine (OPA / Cedar) implementation of `Authorizer`, wired for production-like environments in `provideAuthorizer` (`internal/di/module/authz.go`).

## Notes

- Performs no authorization at all (allows everything).
- Do not use in production.
