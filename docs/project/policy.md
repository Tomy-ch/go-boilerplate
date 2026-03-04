# Project Policy

This document describes the policies used to maintain this boilerplate.

## Maintainer Policy

This repository is an **independent project maintained by the author**.

It is not affiliated with any company or organization.

Architectural and implementation decisions reflect the author's personal design philosophy.

## Disclaimer

This template is provided in good faith.

However, **no guarantees are made regarding suitability for specific use cases, security, or operational stability**.

Users are responsible for verifying:

- Dependency vulnerabilities
- Security configuration
- Operational compatibility

before using this template.

## Maintenance Policy

The maintainer may provide:

- Dependency updates
- Security updates
- Architectural improvements

when possible.

However, the following are **not guaranteed**:

- Response deadlines for Issues
- Guaranteed bug fixes
- Long-term maintenance commitments

If you discover a problem, please open an Issue.

The maintainer will address it when possible.

## Library Selection Policy

The value of this boilerplate lies not in any specific library but in **the integration of widely adopted OSS tools into a coherent architecture**.

Libraries are selected based on the following criteria:

- Actively maintained by a healthy community
- Widely adopted in production environments
- Replaceable if necessary
- Avoiding strong framework lock-in

The architecture assumes **replaceable components**.

Libraries are chosen such that they can be replaced or forked if required.

## Vendor Neutrality

The architecture intentionally avoids tight coupling to specific SaaS vendors.

For example:

- Observability tooling avoids hard dependency on services such as Datadog
- OSS-based tooling is preferred
- Vendor-neutral designs are encouraged

This approach ensures that the system can run in a wide variety of environments.
