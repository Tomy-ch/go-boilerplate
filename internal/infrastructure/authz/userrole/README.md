# userrole (user_roles-based Authorizer)

English | [日本語](README.ja.md)

An `Authorizer` implementation backed by the `user_roles` table. It is the sample counterpart to `allowall`: wired for production-like environments so those environments start with a real (role-based) authorization policy instead of the fail-closed error.

## Role

- Satisfy the `Authorizer` boundary (`internal/usecase/boundary/authz`) using roles assigned to the authenticated subject.
- Decide access from role membership and resource ownership.

## Policy

`Authorize(ctx, authn, action, resource)` decides as follows:

1. The internal UserID must be resolved (`authn.UserID()`); otherwise deny.
2. Fetch the subject's roles via `user.RoleRepository`.
3. If the subject has the admin role (`RoleCodeAdmin`), allow.
4. Otherwise allow only when the subject owns the resource (`subject == Resource.OwnerID()`).
5. Deny otherwise, returning `ErrForbidden` (wraps `apperror.ErrPermissionDenied`, HTTP 403).

Per-API, action-specific authorization is enforced at each usecase; this implementation provides the baseline role/ownership decision.

## DI

Wired in `provideAuthorizer` (`internal/di/module/authz.go`) for the `default` (production-like) environments; `local` / `ci` / `test` keep `allowall`.

## Notes

- Part of the sample domain: removed together with the `user` sample by `make setup-remove-sample-api`, after which `provideAuthorizer` reverts to the fail-closed error.
