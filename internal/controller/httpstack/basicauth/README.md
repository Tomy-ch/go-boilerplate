# basicauth

English | [日本語](README.ja.md)

Provides Basic authentication validator for metrics endpoints.

## Role

The metrics endpoint exposes internal operational data, so it must sit behind an access gate even though it is not part of the business API. This package isolates that credential check as a single reusable validator using a constant-time comparison, keeping the gate policy in one place and out of the endpoint wiring.
