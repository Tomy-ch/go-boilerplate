# testspan

English | [日本語](README.ja.md)

Injects test trace spans into Echo request context.

## Role

Handlers expect a trace span to already exist on the request context because the tracing middleware installs one in production. Tests bypass that middleware, so this helper injects a span deterministically, letting handler tests run the real tracing-dependent code paths without standing up the full middleware stack.
