# Project Policy

This document describes the policies for maintaining and operating this project.

## Maintainer Policy

<!-- boilerplate-only:replace-begin -->
This project is an **independently maintained project managed by the author**.

It is not affiliated with any specific company or organization.

All decisions regarding architecture and implementation are made based on  
**the author's design philosophy**.
<!-- boilerplate-only:replace-with -->
<!-- = Architecture and implementation decisions rest with the team that maintains this repository. -->
<!-- = -->
<!-- = They are made under a single design philosophy rather than per-change preference, so that a -->
<!-- = reader can tell whose judgement the design reflects. -->
<!-- boilerplate-only:replace-end -->

## Disclaimer

This project is provided in good faith.

However, **no guarantees are made** regarding the following:

- Fitness for a particular purpose
- Security
- Operational stability

The following must be verified independently before use:

- Vulnerabilities in dependencies
- Security configurations
- Compatibility with the runtime environment

## Maintenance Policy

<!-- boilerplate-only:replace-begin -->
The maintainer may perform the following within reasonable scope:

- Dependency updates
- Security updates
- Architectural improvements

However, the following are **not guaranteed**:

- Response time to issues
- Bug fixes
- Continuation of long-term maintenance

If you find an issue, please create an Issue.

The maintainer will respond within reasonable scope.
<!-- boilerplate-only:replace-with -->
<!-- = Maintenance covers dependency updates, security updates, and architectural improvements. -->
<!-- = -->
<!-- = What it does not cover — response times, guaranteed bug fixes, how long maintenance -->
<!-- = continues — depends on who maintains the repository and under what commitment. State that -->
<!-- = rather than leave it to assumption. -->
<!-- boilerplate-only:replace-end -->

## Library Selection Policy

The value of this project lies not in specific libraries themselves, but in  
**integrating widely-used OSS tools into a consistent architecture**.

Libraries are selected based on the following criteria:

- Maintained by an active community
- Widely used in production environments
- Replaceable when necessary
- Avoid strong lock-in to specific frameworks

This architecture assumes **component replaceability**.

If necessary, libraries are intended to be replaceable or forkable.

## Vendor Neutrality

This architecture is designed to avoid strong dependency on specific SaaS vendors.

Examples:

- Observability avoids strong dependency on specific services such as Datadog
- OSS-based tools are prioritized
- Vendor-neutral design is encouraged

This policy ensures flexibility to operate the system across various environments.
