# security

English | [日本語](README.ja.md)

Security headers middleware (HSTS, X-Frame-Options, Content-Type-Options, Referrer-Policy).

## Role

Browser-facing hardening headers are a baseline that must apply to every response, not something each handler should remember. Setting them in a single middleware guarantees a uniform security posture across the whole API and keeps the policy defined and auditable in one place.
