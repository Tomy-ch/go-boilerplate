# cookie

English | [日本語](README.ja.md)

Middleware to enforce secure Cookie policies (Secure / HttpOnly / SameSite / Path / Domain / Max-Age).

## Role

Cookie security attributes are easy for individual handlers to forget or set inconsistently. By rewriting outgoing `Set-Cookie` headers in a single middleware, this package guarantees a uniform cookie security policy across every response, so handlers can set cookies without restating the hardening flags each time and the policy stays defined in one place.

## The `ResponseWriter` wrapper must not own its own header map

Rewriting `Set-Cookie` requires wrapping the `ResponseWriter`, and the middleware installs that wrapper for the rest of the request (it is deliberately never restored, so error paths are rewritten too). The wrapper therefore becomes the header map every later participant sees — including the error handler, which reads `X-Request-Id` back off the response to fill the `requestId` of the JSON error body.

So `Header()` **delegates to the wrapped writer** instead of returning a private map. A private map only propagates in the write direction (flushed on `WriteHeader`); values written by middleware that ran *before* the wrapper was installed become unreadable, and the wire header and the response body silently disagree. Any future wrapper added to this stack owes the same guarantee: buffer nothing that a reader can ask for.

## `SECURE_COOKIE_SAME_SITE` clamping (safe default, not silent)

`normalizeSameSite` accepts only `Lax` / `Strict` / `None` (case-insensitive); **any other value — including an empty string — is clamped to "do not override"**, leaving the framework/default `SameSite` in place rather than failing startup. This is a deliberate resilience choice, so a typo in `SECURE_COOKIE_SAME_SITE` weakens the override silently at the value level. It is documented here and enumerated in the setup review ([`docs/get-started/setup-repository.md`](../../../../docs/get-started/setup-repository.md)) so the behavior stays reviewable; set the variable to one of the three accepted values to force a specific policy.
