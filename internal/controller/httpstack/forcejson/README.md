# forcejson

English | [日本語](README.ja.md)

Forces response Content-Type to `application/json`.

## Role

This API contract speaks JSON exclusively, but individual handlers and error paths can leave the response Content-Type unset or inconsistent. Forcing it in a single middleware guarantees every response advertises `application/json`, so clients get a uniform content type without each handler having to set it.
