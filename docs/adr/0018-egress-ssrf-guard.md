---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [http, security, infrastructure]
---

# ADR-0018: Adopt an egress SSRF / dial-guard security posture for outbound HTTP

## Status

accepted

## Context

The application makes outbound HTTP calls to URLs that are at least partially under operator
or user influence: gateway endpoints configured at deploy time, webhook receiver URLs stored
in the database, and the outbox publisher endpoint. If any such URL can be pointed at an
internal IP address, an attacker can leverage the server as a proxy to access internal
services, cloud metadata endpoints, or other resources that are unreachable from the public
internet. This class of vulnerability is Server-Side Request Forgery (SSRF).

Key threat vectors specific to this application:

**Cloud metadata service.** Cloud providers typically expose instance metadata (including
temporary credentials) at `169.254.169.254` (link-local unicast). A URL like
`http://169.254.254.169/latest/meta-data/iam/security-credentials/` would return credentials
if reached. Standard HTTP libraries do not block this address.

**DNS rebinding.** A DNS response may resolve a hostname to a private IP that was not
intended. Blocking a hostname at URL-parse time is insufficient because the IP seen at
connection time may differ. The guard must run post-DNS, at dial time, against the resolved IP.

**Private-network access from misconfigured operator URLs.** An operator who supplies a
webhook URL might accidentally (or intentionally) provide an RFC1918 address. Without a guard,
the application silently connects to internal services.

**Redirect-based SSRF.** A public endpoint may respond with a redirect to a private IP.
Following redirects without validation extends the SSRF surface.

Go's standard `net/http` does not guard against any of these vectors out of the box. Relying
solely on network-layer egress controls (firewall, security-group rules) provides defence in
depth but does not prevent DNS rebinding and is not visible at the application layer.

## Decision

All outbound HTTP transport runs through `internal/observability.HTTPClientTransport`, which
wraps the base `*http.Transport` with a `net.Dialer` whose `ControlContext` is set to
`guardedDialControl`. The guard runs **post-DNS at dial time** — it inspects the resolved IP
address in the `address` argument passed by the kernel after name resolution, not the original
hostname in the URL — which also prevents DNS rebinding attacks.

The guard enforces a two-tier blocking policy:

**Always blocked (regardless of configuration):**

- Link-local unicast and link-local multicast (`169.254.0.0/16`, `fe80::/10`) — cloud
  metadata services and similar internal endpoints.
- Unspecified address (`0.0.0.0`, `::`) — not a valid destination.
- Bogon / reserved ranges that are never legitimate public destinations: Future Use
  (`240.0.0.0/4`), IETF Protocol Assignments (`192.0.0.0/24`), TEST-NET-1/2/3
  (`192.0.2.0/24`, `198.51.100.0/24`, `203.0.113.0/24`), Benchmarking (`198.18.0.0/15`),
  IPv6 Documentation (`2001:db8::/32`).

**Blocked by default, allow-listable per downstream:**

- Loopback (`127.0.0.0/8`, `::1`)
- Private networks (RFC1918 `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`; ULA
  `fc00::/7`)
- CGNAT shared address space (RFC 6598 `100.64.0.0/10`) — Go's `net.IP.IsPrivate` does not
  cover CGNAT, so it is checked explicitly.

A downstream that legitimately calls internal services (e.g. an internal gateway) sets
`AllowPrivateNetwork = true` in its `Profile`. External-facing downstreams (e.g. the outbox
publisher) set `AllowPrivateNetwork = false` explicitly. The flag is propagated from the
`Profile` into a context value (`allowPrivateNetworkKey`) per attempt by the
`httpclient.Client`, so each attempt's dial inherits the correct posture.

In addition, redirects are not followed (`http.ErrUseLastResponse` on `CheckRedirect`),
preventing redirect-based SSRF where a public endpoint redirects to a private IP.

The `HTTPClientTransport` also wraps the base transport with OpenTelemetry instrumentation
(`otelhttp`) and a conditional trace-propagation filter, so the guard is wired in as part of
the single canonical transport used by all outbound calls (see
[ADR-0017](0017-outbound-http-resilience.md)).

## Consequences

### Positive Consequences

- The guard runs post-DNS, so DNS rebinding attacks are blocked even when the hostname
  appears benign at URL-parse time.
- All outbound calls share one transport instance, so the guard cannot be accidentally omitted
  by a new gateway or publisher implementation.
- Link-local (`169.254.x.x`) and bogon-reserved ranges are blocked unconditionally; no
  misconfiguration can re-enable them.
- The `AllowPrivateNetwork` flag is per-downstream and defaults to `true` for internal
  services, so internal-to-internal calls work without manual allowlisting.
- No redirect-following prevents a second class of SSRF (redirect chains to internal IPs).

### Negative Consequences

- A new gateway that calls an internal service must remember to set `AllowPrivateNetwork =
  true` in its `DownstreamProfile`; forgetting causes dial failures against internal IPs.
- The guard fires synchronously at dial time; a blocked dial results in an error that the
  `httpclient` substrate normalises to `ErrInvalidArgument`. Callers must treat this case as a
  configuration error rather than a transient failure.
- Testing outbound calls against local test servers requires either the permissive dial
  control (`permissiveDialControl`, available only in test builds) or setting
  `AllowPrivateNetwork = true` in the test profile.

## Alternatives Considered

### Host / IP allow list only

Permit only explicitly approved hosts or IP ranges; reject everything else. Provides a tighter
posture. Rejected because it requires updating the allow list for every new downstream and
makes the template harder to adopt in diverse environments. The deny-list approach (block known
bad ranges, allow the rest) is more operationally practical while still covering the primary
SSRF targets.

### Network-level egress controls only (firewall / security group)

Cloud security groups and host-based firewalls can restrict outbound connections. Rejected as
the sole control because: (a) firewall rules do not see DNS rebinding — the IP at connection
time may differ from what the firewall pre-approved; (b) they are infrastructure-layer
controls invisible to the application, making it harder to audit application-level security
intent; (c) defence in depth is better than a single control point.

### No redirect guard

Allow `http.Client` to follow redirects (default behaviour). Rejected because a public HTTP
endpoint may issue a 3xx redirect to a private IP, bypassing the dial guard for the initial
request. Returning the last response (`http.ErrUseLastResponse`) forces the caller to decide
whether to follow the redirect, keeping the decision at the application layer.

### Per-gateway custom transport

Each gateway constructs its own transport with or without a guard. Rejected because it creates
a footgun: a new gateway that omits the guard silently opens an SSRF surface. The single
canonical transport in `HTTPClientTransport` makes it structurally impossible to bypass the
guard.

## Notes

- Guard implementation: `internal/observability/http_client_transport.go`
  (`guardedDialControl`, `reservedNets`, `cgnatNet`, `allowPrivateNetworkFromContext`).
- Transport construction: `NewHTTPClientTransport` wires `guardedDialControl` into the base
  transport and wraps it with `otelhttp`.
- Consumer that sets `AllowPrivateNetwork = false` for an external downstream:
  `internal/infrastructure/publisher/http_publisher.go` (`NewDownstreamProfile`).
- Resilience substrate that propagates the `AllowPrivateNetwork` context flag per attempt:
  `internal/infrastructure/httpclient/client.go` (`attempt`).
- Related: [ADR-0017](0017-outbound-http-resilience.md) (resilience substrate that hosts
  this transport).
- source: `internal/observability/http_client_transport.go`;
  `internal/infrastructure/httpclient/README.md` (§ "Design Policy", security defaults).
