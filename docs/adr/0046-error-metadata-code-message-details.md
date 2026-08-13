---
status: accepted
date: 2026-07-16
deciders: [maintainers]
tags: [errors, architecture, api]
---

# ADR-0046: Protocol-neutral error metadata (code / message / details) on top of apperror

## Status

accepted

## Context

[ADR-0045](0045-apperror-protocol-agnostic-errors.md) established the protocol-agnostic
error taxonomy: sentinels in `internal/apperror` classify failures, and the controller's
error-handler middleware maps each sentinel to a fixed HTTP status with a fixed error
`code` and `message`. This mapping is strictly 1:1 per status, so an error-raising site
cannot communicate anything dynamic to the API client — for example, *which* fields of a
user-update request failed validation.

The concrete need: `PUT/PATCH /v1/users/{id}` should report the invalid fields in the
response (`details: ["firstName", "email"]`) **without** exposing the reason each field
failed (reasons remain log-only). Additionally, domain validation collected only the
first failing field (`validateProfileFields` returned on the first error), so multiple
simultaneous violations could not be reported together.

## Decision

`internal/apperror` gains a **decoration** type, orthogonal to the classification
sentinels:

- `Meta{Code, Message, Details}` — all fields optional, protocol-neutral.
- `WithMeta(err, meta)` / `WithDetails(err, details...)` attach a `Meta` to an error
  chain; `MetaFrom(err)` extracts the **outermost** one via `xerrors.As`.
- The wrapper (`MetaError`) implements `Unwrap`, so `xerrors.Is` / `IsAppError` still
  see the wrapped sentinel(s) — including every branch of a `xerrors.Join`.

At the edge, `NewHTTPErrorFromAppError` overlays a present `Meta` on the defaults
resolved from the sentinel classification: a non-empty `Code` / `Message` overrides the
status default; `Details` is used when no explicit details argument is given.

Three constraints keep ADR-0045 intact:

1. **`Meta` never carries an HTTP status.** The status is resolved solely from the
   sentinel classification; to change the status, change the sentinel. Allowing a
   status in `Meta` would re-introduce the HTTP-status-carrying error type that
   ADR-0045 explicitly rejected.
2. **`Message` stays under controller ownership.** The field exists for generality,
   but domain / usecase leave it empty; user-facing wording remains centrally managed
   in the controller catalog, so API wording changes never touch inner layers.
3. **`Details` holds public-safe identifiers only** (e.g., invalid field names as
   domain constants matching the API property names — `user.FieldFirstName`), never
   reason texts or raw input values. Reasons stay in the wrapped error message
   (`xerrors.Wrap(ErrInvalidXxx, msg)`), which is surfaced only in logs.

Supporting change: `user.validateProfileFields` now validates **all** profile fields
and joins the per-field sentinel errors (`xerrors.Join`), attaching the collected field
identifiers via `WithDetails`. Server-internal invariants (id, updatedAt, deletedAt)
keep first-error return — they are not user-correctable input. As a
side effect, `POST /v1/users` (creation) also reports invalid fields in `details`;
this is an intended improvement.

## Consequences

### Positive Consequences

- Error-raising sites can return dynamic, machine-readable specifics (invalid field
  lists, feature-specific codes) without adding a sentinel or touching the edge mapping.
- Multiple simultaneous validation failures are reported in one response instead of
  one-at-a-time round trips.
- The response contract stays backward compatible: `details` is an existing optional
  field; status / default code / default message are unchanged.
- Reasons and values never leak: the public surface is identifiers only.

### Negative Consequences

- Joined validation errors make `err.Error()` multi-line; tests comparing full error
  strings (rather than `xerrors.Is`) would need adjustment.
- The outermost-wins extraction rule means an upper layer re-attaching `Meta` hides the
  inner one — intentional, but it requires awareness when wrapping.
- The "identifiers only" rule for `Details` is enforced by convention and review, not
  by the type system.

## Alternatives Considered

### HTTP-status-carrying metadata

Let `Meta` override the HTTP status. Rejected: re-introduces the coupling ADR-0045
removed, and allows status/code inconsistencies (e.g., 422 with `NOT_FOUND`) that the
sentinel-only resolution structurally prevents.

### Controller-side sentinel→field-name mapping table

Keep domain free of field identifiers and translate `ErrInvalidXxx` sentinels to field
names at the edge. Rejected: the generic error handler would need to probe every
domain's sentinel list (`xerrors.Is` fan-out over joined errors), accumulating domain
knowledge and growing with every aggregate. Field identifiers are domain vocabulary
(not wording, not protocol), so the domain exposing them as constants is acceptable;
if API names ever diverge, the controller can re-map at that point.

### Per-field sentinel + fixed code for every case

Add an apperror sentinel and a fixed code/message per dynamic case. Rejected: constant
explosion for field-level reporting, and still cannot express request-specific data.

## Notes

- Source: `internal/apperror/README.md` — Error Metadata (`Meta`) section;
  `internal/controller/error/response/README.md` — `apperror.Meta` Overrides section.
- Prefecture-name resolution failures on user update remain `ErrNotFound` (404) via the
  repository path and are out of scope for field `details`; folding them into 422 is a
  separate decision if ever needed.
- PATCH validates the merged full profile, so in theory a field the client did not send
  can appear in `details` if stored data drifted from the invariants; not expected in
  practice.
