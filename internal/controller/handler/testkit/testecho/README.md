# testecho

English | [日本語](README.ja.md)

Builder-pattern HTTP test client for Echo handler tests.

## Role

Driving a handler through the HTTP boundary requires assembling a request, response recorder, and request context the same way in every test. This builder hides that repetitive setup behind a fluent API, so handler tests construct a request in a few readable lines and exercise the real HTTP entry path consistently.
