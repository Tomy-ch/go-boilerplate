# forcejson

English | [日本語](README.ja.md)

Forces the response Content-Type to `application/json` when it is unset or `text/html`.

## Role

This API contract speaks JSON exclusively, but individual handlers and error paths can leave the response Content-Type unset or defaulted to `text/html`. A single middleware normalizes those cases to `application/json` (an already-explicit Content-Type such as `text/csv` is left untouched), so JSON responses advertise a uniform content type without each handler having to set it.
