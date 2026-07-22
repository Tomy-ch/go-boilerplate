# oapi/auth

English | [日本語](README.ja.md)

OpenAPI authentication function that extracts a Bearer token from the `Authorization` header, validates it via boundary `Authenticator`, and stores the result in the request context (an authn slot). Cookie-based extraction is not supported (Bearer / Resource Server model).

## Token Extraction Flow

```mermaid
flowchart TB
    Start["Request"]
    IsBearer{"Authorization: Bearer <token>?"}
    StripBearer["Strip 'Bearer ' prefix → scheme=Bearer"]
    NoToken["Token empty → ErrUnauthorizedTokenNotProvided"]
    Credential["NewCredential(scheme, token)"]
    Authenticate["authenticator.Authenticate(ctx, credential)"]
    StoreAuthn["ctxhelper.SetAuthn(req.Context(), authn)"]

    Start --> IsBearer
    IsBearer -- yes --> StripBearer --> Credential
    IsBearer -- no --> NoToken
    Credential --> Authenticate --> StoreAuthn
```

### Extraction Rules

1. **Header** — Extract from the fixed `Authorization` header. Bearer tokens are pinned to `Authorization` by RFC 6750, so the header name is not configurable
2. **Bearer prefix** — Only `Authorization: Bearer <token>` is accepted; the `Bearer` prefix is stripped and the credential scheme becomes `Bearer`. Any other form yields no token
3. If no token is found, return `ErrUnauthorizedTokenNotProvided`

### Authentication Steps

1. Extract the Bearer token from the `Authorization` header (rules above)
2. Create `boundary/auth.Credential` from the scheme and token
3. Call `authenticator.Authenticate(ctx, credential)` to obtain `Authn`
4. Store `Authn` into the request context via `ctxhelper.SetAuthn()` (the slot is seeded upstream by `ctxhelper.WithAuthn` in `oapi.Middleware`); returns `ErrAuthnSlotNotFound` if the slot is missing

Handler code can then retrieve `Authn` using `ctxhelper.GetAuthn()`.

## Errors

|Error|Base Error|Description|
|---|---|---|
|`ErrUnauthorizedInvalidToken`|`ErrUnauthenticated`|Token validation failed by `Authenticator`|
|`ErrUnauthorizedTokenNotProvided`|`ErrUnauthenticated`|No token found in the `Authorization` header|
|`ErrUnauthorizedTokenMissing`|`ErrUnauthenticated`|Authorization token is missing|
|`ErrAuthnSlotNotFound`|`ErrUnauthenticated`|Authn slot not found in the request context (slot not seeded by `oapi.Middleware`)|
|`ErrInvalidAuthDefaultMode`|`ErrInternal`|Default auth policy not found|

## Authn Slot Integration

This function runs inside the OpenAPI validation pipeline, where only `context.Context` is available (not `echo.Context`). The parent `oapi.Middleware` seeds an **authn slot** into `request.Context()` (via `ctxhelper.WithAuthn`) before validation runs, so the authFunc — invoked by the validator — can write the authenticated `Authn` back into that slot with `ctxhelper.SetAuthn`. The handler later reads it with `ctxhelper.GetAuthn`.

```mermaid
flowchart LR
    OapiMW["oapi.Middleware"] -->|"WithAuthn (seed slot)"| ReqCtx["request.Context()"]
    ReqCtx --> Validator["oapi validator → authFunc"]
    Validator -->|"SetAuthn"| ReqCtx
    Handler["handler"] -->|"GetAuthn"| ReqCtx
```

## Notes

- Token extraction is header-only; cookies are not consulted (Bearer / Resource Server model)
- Only the `Authorization: Bearer <token>` form is accepted (RFC 6750); non-Bearer schemes and custom header names are not supported
- The `Authenticator` implementation is environment-specific (local mock, JWT, OAuth, etc.) and injected via DI
