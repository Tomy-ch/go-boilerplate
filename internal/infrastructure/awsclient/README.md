# infrastructure/awsclient

English | [日本語](README.ja.md)

## Role

Builds the `aws.Config` that every AWS SDK v2 adapter in this repository starts from — currently the
SQS queue adapter and the S3 object-storage adapter. It owns two decisions those adapters would
otherwise each answer for themselves: **where credentials come from**, and **which HTTP client each
kind of request uses**.

## Credentials: one path, not a choice

`Resolve` does not offer a "static or chain" switch. The SDK's default credential provider chain
already **contains** static credentials — the environment-variable provider is one of its links,
alongside the shared profile, web identity, and the container / IMDS providers. There is no
static-versus-chain fork in the SDK's own model, so adding a discriminator would re-invent a
distinction the standard has already settled.

Explicitly injected credentials are therefore treated as an **override of the chain**, not as an
alternative to it:

| `AccessKeyID` / `SecretAccessKey` | Behaviour |
| --- | --- |
| both set | overrides the chain with those static credentials |
| both empty | resolves through the default chain (IAM role, environment, shared profile, …) |
| exactly one set | rejected with `ErrInvalidCredentials` |

The half-set case is rejected because it reads as neither intent, and because the SDK will happily
sign a request with an access key ID and an empty secret — the failure would surface as an
authentication error on the first API call rather than at startup.

## Fail-fast

`Resolve` retrieves credentials once before returning. A deployment whose credentials cannot be
resolved fails at startup instead of at the first call, which is the same reason the outbox publisher
validates its endpoint eagerly: a relay that starts and then fails every publish burns through
attempts and lets messages go dead unnoticed.

This is what replaced the older "reject empty credentials" check. That check could not tell a
misconfiguration from a deliberate hand-off to the chain; asking whether credentials **resolve**
answers both at once.

## Two HTTP clients, because they guard different things

The `HTTPClient` on `Config` is applied to **service API calls only**. Credential resolution runs on
the SDK's own transport.

The application's outbound client (`observability.OutboundHTTPClient`) refuses link-local
destinations unconditionally, so that an endpoint pointed at cloud metadata is denied at dial time.
But EC2's IMDS (`169.254.169.254`) and ECS's task metadata (`169.254.170.2`) are link-local by
definition. Sharing one client between the two purposes would leave exactly the most common
role-based deployments unable to obtain credentials at all.

They are different concerns: the guard exists to constrain **egress to external services**, while
credential resolution is the process asking **its own platform** who it is. Splitting them keeps the
guard on the traffic it was written for.

The exemption is wider than the metadata endpoints, and worth stating plainly: **every**
credential-resolution request runs on the SDK's transport — STS web-identity exchange (EKS IRSA)
and SSO token exchange included. Neither the destination-IP check nor the guard's proxy-disabling
(`Proxy = nil`) applies to them.

That is deliberate, not an oversight waiting to be closed. Redirecting this traffic
(`AWS_EC2_METADATA_SERVICE_ENDPOINT`, `AWS_CONTAINER_CREDENTIALS_FULL_URI`, `HTTPS_PROXY`) takes
write access to the process environment, and anything that can set those can read the credentials
directly. The SDK also guards the one env-overridable plaintext endpoint itself, restricting it to
loopback and the known ECS / EKS addresses. And the chain could not be covered even if we tried:
the container-credential provider does not inherit the client passed to `LoadDefaultConfig`, so a
guarded transport would reach IMDS, STS, and SSO but never that path.

The full reasoning, including why a link-local-permitting variant of the guard was rejected, is
recorded in
[ADR-0022](../../../docs/adr/0022-egress-ssrf-guard.md#guarding-the-aws-credential-chain).

## Notes

- `Region` may be left empty to let the SDK resolve it (environment, shared profile). The adapters
  above pass their own configured region.
- Endpoint overrides stay in each adapter, since they are service-specific
  (`sqs.Options.BaseEndpoint`, `s3.Options.BaseEndpoint` + `UsePathStyle`).
