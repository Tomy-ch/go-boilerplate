# testkit

English | [日本語](README.ja.md)

Test utilities for Controller layer tests.

## Role

Controller-layer tests repeatedly need the same scaffolding: a configured HTTP test client, an authenticated request context, an injected trace span, and structured response assertions. Collecting these into shared helpers removes that boilerplate and provides deterministic test doubles, so handler tests stay short, consistent, and focused on the behaviour under test rather than on setup.

## Subpackages

|Package|Description|
|---|---|
|`testassert`|JSON response and Echo router assertions|
|`testauth`|Test authentication context setup|
|`testecho`|Echo test client builder|
|`testspan`|Test span injection for Echo context|
