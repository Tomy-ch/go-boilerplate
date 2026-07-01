# testassert

English | [日本語](README.ja.md)

Assertion helpers for Controller layer tests.

## Role

Verifying JSON responses and HTTP routing by hand means repeating unmarshal-and-compare and route-lookup logic in every test, which is verbose and easy to get subtly wrong. These helpers centralize those assertions so handler tests express expectations declaratively and report mismatches consistently in one place.
