# cookie

English | [日本語](README.ja.md)

Middleware to enforce secure Cookie policies (Secure / HttpOnly / SameSite / Path / Domain / Max-Age).

## Role

Cookie security attributes are easy for individual handlers to forget or set inconsistently. By rewriting outgoing `Set-Cookie` headers in a single middleware, this package guarantees a uniform cookie security policy across every response, so handlers can set cookies without restating the hardening flags each time and the policy stays defined in one place.
