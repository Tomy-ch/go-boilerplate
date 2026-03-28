# Project Policy

This document describes the policies for maintaining and operating this project.

## Maintainer Policy

This project is an **independently maintained project managed by the author**.

It is not affiliated with any specific company or organization.

All decisions regarding architecture and implementation are made based on  
**the author's design philosophy**.

## Disclaimer

This template is provided in good faith.

However, **no guarantees are made** regarding the following:

- Fitness for a particular purpose
- Security
- Operational stability

When using this template, users must verify the following by themselves:

- Vulnerabilities in dependencies
- Security configurations
- Compatibility with their runtime environment

## Maintenance Policy

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
