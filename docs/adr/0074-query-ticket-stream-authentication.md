---
status: accepted
date: 2026-08-28
deciders: [maintainers]
tags: [security, http, realtime, contract]
---

# ADR-0074: Authenticate SSE streams with an opaque query ticket bound to subject, destination, scope, and expiry

## Status

accepted

## Context

The browser's `EventSource` API cannot set request headers. The bearer-token scheme every other
endpoint uses ([ADR-0016], [ADR-0021]) is therefore unavailable to an SSE connection at the moment
it matters most — the connect. The alternatives each carry a cost that has to be weighed rather than
assumed away: a cookie session drags CSRF, `SameSite`, and CORS design into a mechanism that is
otherwise stateless; a bearer token in the query string puts a long-lived credential into URLs,
access logs, and `Referer` headers; a one-time token breaks the browser's own reconnect, which
re-sends the same URL.

Authorization for a stream also has a lifetime problem that a request-scoped check never had. A
REST request is authorized once and is over in milliseconds. A stream stays open for an hour, and
during that hour the subject's access to the destination can be withdrawn — by this service
(membership invalidated, a conversation's access revoked) or by the identity provider (account
deactivated). Whatever authenticates the connect must say what happens after it.

## Decision

A feature endpoint, after performing its own authorization, issues an **opaque 256-bit stream
ticket**. The ticket is stored only as a hash and is bound to four things: the **subject** (a
feature-neutral principal identifier — the stream runtime never learns whether it is a user or an
operator), the **destination** (the stream it may read), a **scope**, and an **expiry**. The client
presents it as a query parameter when opening the stream.

- **Reusable for its TTL, not one-time.** New connections are accepted for 5 minutes after issue,
  so the browser's automatic reconnect — same URL, `Last-Event-ID` set — works unchanged.
- **Ticket TTL and connection lifetime are different quantities.** The TTL bounds when a *new*
  connection may start. An established connection has its own **maximum lifetime of one hour**,
  after which the server closes it with a `REAUTHENTICATE` control event; the client obtains a fresh
  ticket through the feature's authorization path and reconnects. Both values are fixed, not
  deployment configuration.
- **Never logged, never traced.** The raw ticket is excised from request URIs, query parameters,
  error and recovery logs, and span attributes before any of them are emitted. This is enforced in
  the HTTP stack, not left to each handler.
- **Revocation inside this service is immediate.** When this service withdraws access — a
  membership soft-delete, which the identity resolver already enforces per request for REST
  ([ADR-0021]), or a feature revoking a subject's access to a destination — the feature calls a
  revocation seam in `boundary/realtime`. The seam invalidates every ticket held by that subject
  for that destination and notifies every serve instance through the existing fan-out
  ([ADR-0073]); each instance closes the matching connections with a `STOP` control event. The
  connection registry is indexed by subject for this purpose. Because the ticket is invalidated
  too, a client that ignores `STOP` cannot reconnect with it.
- **Revocation at the identity provider is not observed.** An account deactivated at the IdP
  keeps its already-issued JWTs valid until they expire, and this service does not poll the IdP;
  that is the existing authentication posture, not a new one. An open stream therefore converges on
  such a change only through the one-hour maximum lifetime, exactly as a REST bearer token converges
  through `exp`. A thin IdP-to-service ingress adapter, when one is added, calls the same revocation
  seam and turns that into an immediate close.

## Consequences

### Positive Consequences

- Works with the platform `EventSource` and its built-in reconnect; no polyfill and no custom
  transport on the client.
- The credential in the URL is short-lived, scoped to one stream, stored hashed, and useless after
  revocation — the properties a bearer token in a query string lacks.
- Authorization stays where it is today: in the feature's usecase at ticket issue, and in the
  identity resolver per request. The stream runtime performs no authorization of its own and holds
  no feature vocabulary to do it with.
- Withdrawal of access inside this service reaches an open stream in seconds, not at the next
  reconnect.

### Negative Consequences

- Every feature that streams must own a ticket-issuing endpoint and decide its own scope; there is
  no generic "issue me a ticket" operation.
- A connection can outlive an IdP-side deactivation by up to an hour. The bound is explicit and
  matches the REST token bound, but it is a bound, not zero.
- Excising the ticket from logs and traces is a rule applied to shared middleware; a new logging
  path that forgets it leaks the credential, and the tests that pin the excision are the only guard.
- The ticket store is one more table with a TTL, and revocation adds a fan-out message type to the
  wakeup topic.

## Alternatives Considered

### `Authorization: Bearer` on the stream request

Unavailable: `EventSource` sets no headers. Requiring a fetch-based client to add one pushes a
custom transport onto every consumer.

### A cookie-backed session for the stream endpoint

Rejected. It adds CSRF, `SameSite`, and CORS credential handling specific to SSE, and it duplicates
the session the BFF in front of this service already holds.

### A one-time ticket

Rejected. The browser reconnects with the same URL; a consumed ticket turns every automatic
reconnect into a `401` and defeats `Last-Event-ID` resume.

### The JWT itself as a query parameter

Rejected. A long-lived, replayable credential in URLs, access logs, and `Referer` headers, with no
way to revoke it short of key rotation.

### Rely on the connection lifetime alone for revocation

Rejected. It leaves a revoked subject receiving events for up to an hour when the revocation
happened in this very service, where an immediate close costs one fan-out message.

### Poll the identity provider from the stream runtime

Rejected. It contradicts the local-evaluation model the authentication design already commits to,
adds a per-connection dependency on the IdP's availability, and would still miss the interval
between polls.

## Notes

- Design reference: `docs/design/realtime-delivery.md` §2 (ticket and connection lifecycles) and
  §4 (the client contract for `REAUTHENTICATE` and `STOP`); `docs/design/auth.md` §2 for the two
  invalidation axes this decision inherits; `docs/design/security.md` for where the log excision
  fires.
- Related: [ADR-0071], [ADR-0073], [ADR-0016] (spec-driven auth: the query-ticket security scheme
  is declared in OpenAPI and dispatched beside the bearer scheme), [ADR-0021] (a failed
  authentication denies the request — a bad or revoked ticket is refused before the response is
  committed), [ADR-0108] (reconnect *rate* is an edge concern; the ticket bounds *who*, not
  *how often*).

[ADR-0016]: 0016-spec-driven-request-validation.md
[ADR-0021]: 0021-optional-authentication-fail-closed.md
[ADR-0071]: 0071-realtime-delivery-driving-mechanism.md
[ADR-0073]: 0073-sns-sqs-instance-fanout.md
[ADR-0108]: 0108-no-in-app-rate-limiter.md
