# testuuid

English | [日本語](README.ja.md)

Generates openapi UUIDs for handler request tests.

## Role

Handler request tests need UUID values in the API's generated type, and constructing them inline pulls in conversion plumbing that obscures intent. This helper produces a ready-to-use value in one call, so request fixtures stay concise and tests do not couple to UUID-construction details.
