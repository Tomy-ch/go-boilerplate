# architest

English | [日本語](README.ja.md)

`internal/architest` holds **only tests**: machine-verified checks of cross-cutting invariants that no
single package owns — layer dependency rules, and agreement between implementation, data and
documentation across the repository. It contains no production code.

## Role

A regular package's tests verify that package's behaviour. The invariants here have no such owner: they
hold *between* packages, or between code and a file that is not code at all (an OpenAPI spec, an `env`
file, a Dockerfile, a README table). Left to review, each one fails silently — the repository stays
green while a route quietly stops being served, or a timezone is propagated to three of four places.

Because the subject is a contract rather than a function, these tests have no production counterpart, and
the one-directional 1:1 mapping in [`docs/testing-conventions.md`](../../docs/testing-conventions.md)
makes no claim about them.

## Test Strategy

This package is not a layer, so no layer README governs it; per section 11 of
[`docs/testing-conventions.md`](../../docs/testing-conventions.md) its viewpoints live here. The
cross-cutting structure rules (`t.Parallel()`, subtest groups, assertions) still come from that document
— only the viewpoints below are local.

- **Name the sets being reconciled, and what each direction detects.** A contract test asserts a relation
  between two or more sets (`X == Y`, `Z ⊆ X`). Each direction catches a *different* mistake and is worth
  stating separately: one direction is usually the reason the test exists, while the reverse direction
  exists to fail loudly if the scan itself silently shrinks. A test that only says "these must match"
  leaves the next reader unable to tell which failure they are looking at.
- **Pin the scanning logic against synthetic sources.** A collector whose matching is non-trivial gets a
  companion `Test_collectXxx` / `Test_scanXxx` driving it over inline sources rather than the real tree.
  This is the viewpoint that keeps the others honest: a scanner that stops recognising a syntax shape
  reports "no violations" and stays green forever, so the scan's own coverage cannot be left to the
  repository's current contents. The real-tree walk that finds files stays outside this viewpoint: it
  reads the filesystem and has no syntax shape to recognise, so what gets a companion is the pure
  function it hands each file's lines to.
- **Report violations with their declaration site, in a stable order.** Findings carry the file (and line
  where meaningful) that declared them and are sorted before assertion, so a failure is deterministic and
  points at what to edit rather than at the invariant in the abstract. Collecting into a slice and
  asserting once also reports every violation together, which iterating a map with a per-item assertion
  cannot do in a reproducible order.
- **Pin each deliberate carve-out as a case of its own.** Generated files, `main` / `init`, a test file
  excluding itself from its own scan, and documented `t.Skip` forms are all *expected* not to trip a
  check. Asserting that explicitly is what distinguishes a carve-out from a hole opened later by
  accident.

Allowlists are deliberately avoided: an allowlist is itself a drift source, so an exception is either
expressed as a structural carve-out with a test pinning it, or it is a failure.

## Notes

- `depguard` forbids `go/ast` here, so detection is text scanning over gofmt-normalised sources.
  Assuming gofmt's output (indentation depth, `var (` blocks at column 0) is what makes that tractable.
- A check may legitimately match zero subjects — the sample API can be removed. The scan-logic tests
  above are what keep a zero count from being indistinguishable from a broken scanner.
