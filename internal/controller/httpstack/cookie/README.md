# cookie

English | [日本語](README.ja.md)

Middleware to enforce secure Cookie policies (Secure / HttpOnly / SameSite / Path / Domain / Max-Age).

## Role

Cookie security attributes are easy for individual handlers to forget or set inconsistently. By rewriting outgoing `Set-Cookie` headers in a single middleware, this package guarantees a uniform cookie security policy across every response, so handlers can set cookies without restating the hardening flags each time and the policy stays defined in one place.

## `SECURE_COOKIE_SAME_SITE` clamping (safe default, not silent)

`normalizeSameSite` accepts only `Lax` / `Strict` / `None` (case-insensitive); **any other value — including an empty string — is clamped to "do not override"**, leaving the framework/default `SameSite` in place rather than failing startup. This is a deliberate resilience choice, so a typo in `SECURE_COOKIE_SAME_SITE` weakens the override silently at the value level. It is documented here and enumerated in the setup review ([`docs/get-started/setup-repository.md`](../../../../docs/get-started/setup-repository.md)) so the behavior stays reviewable; set the variable to one of the three accepted values to force a specific policy.
