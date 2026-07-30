# userrole (user_roles-based Authorizer)

English | [日本語](README.ja.md)

An `Authorizer` implementation backed by the `user_roles` table. It is the sample counterpart to `allowall`: wired for production-like environments so those environments start with a real (role-based) authorization policy instead of the fail-closed error.

## Role

- Satisfy the `Authorizer` boundary (`internal/usecase/boundary/authz`) using roles assigned to the authenticated subject.
- Decide access from role membership and resource ownership.

## Policy

`Authorize(ctx, authn, action, resource)` decides as follows:

1. The internal UserID must be resolved (`authn.UserID()`) and non-zero; otherwise deny. A zero-value UUID does not identify a resolved subject, and step 4 compares values only — letting it through would make a zero-value subject match a zero-value owner.
2. Fetch the subject's roles via `user.RoleRepository`.
3. If the subject has the admin role (`RoleCodeAdmin`), allow.
4. Otherwise allow only when the subject owns the resource (`subject == Resource.OwnerID()`). A resource whose `OwnerID()` is `nil` has no owner to compare against, so this fallback can never succeed and only step 3 can allow — the action is effectively **admin-only**. Building a `Resource` without an owner is therefore how a caller declares an admin-only operation, and it fails in the safe direction: omitting an owner narrows access, never widens it.
5. Deny otherwise, returning `ErrForbidden` (wraps `apperror.ErrPermissionDenied`, HTTP 403).

Per-API, action-specific authorization is enforced at each usecase; this implementation provides the baseline role/ownership decision.

## DI

Wired in `provideAuthorizer` (`internal/di/module/authz.go`). While the `user` sample is present, `ci` / `test` receive `allowall` and every other known environment — `local`, `development`, `staging`, `production` — receives this implementation, so a local run exercises the real role-based decision. Removing the sample folds `local` into the `allowall` case and reverts the production-like environments to the fail-closed error.

## Notes

- Part of the sample domain: removed together with the `user` sample by `make setup-remove-sample-api`, after which `provideAuthorizer` reverts to the fail-closed error.
