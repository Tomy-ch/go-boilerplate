# cors

English | [日本語](README.ja.md)

CORS middleware configured from security settings.

## Role

Cross-origin access rules are a deployment-wide policy, not a per-endpoint concern. Isolating CORS into a configuration-driven middleware keeps the allowed origins, methods, and headers defined once from security settings and applied uniformly, so handlers never deal with preflight or origin negotiation.
