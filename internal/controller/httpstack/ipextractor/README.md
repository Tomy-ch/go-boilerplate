# ipextractor

English | [日本語](README.ja.md)

Configures client IP extraction strategy based on environment.

## Role

The real client IP is derived differently depending on whether the app runs directly exposed or behind a trusted proxy or load balancer, and getting it wrong undermines logging and any IP-based control. Centralizing the extraction strategy as an environment-driven setup gives the rest of the stack a single, correctly-derived client IP to trust.
