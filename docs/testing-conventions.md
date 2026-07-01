# Testing Conventions

This is the **canonical source for how tests are written** in this repository — structure,
naming, parallelism, assertions, mocks, and coverage-exception governance. It is read at
runtime by the `scaffold-test` (generation) and `test-review` (review) skills, so keeping a
single source here prevents drift.

Scope split (do not duplicate across these):

- **This document** — the concrete *how* (techniques / conventions).
- [`rules.md` → *Testing & Definition of Done*](rules.md) — the non-negotiable *when is it done* (per-layer testing, the 90 % bar, "compiles ≠ done", runtime DI verification, live-app smoke tests, unreachable-branch policy).
- Each layer `README` → *Test Strategy* — the per-layer **viewpoints** (what to exercise for that layer).

The canonical reference test is [`internal/domain/user/user_domain_test.go`](../internal/domain/user/user_domain_test.go).

## 1. Structure

- **One `TestXxx` per function or method.** Bundling multiple subjects into one test function requires explicit case-by-case justification.
- Every logical branch is exercised.
- The **outermost `t.Run` groups are the literal strings `正常系` / `異常系`** — not a `正常系_xxx` prefixed form. Nest further `t.Run` subcases inside them.

```go
func TestNewUser(t *testing.T) {
    t.Parallel()
    t.Run("正常系", func(t *testing.T) {
        t.Parallel()
        t.Run("全ての入力が正しい場合、エンティティが生成される", func(t *testing.T) { /* ... */ })
    })
    t.Run("異常系", func(t *testing.T) {
        t.Parallel()
        t.Run("IDがゼロ値の場合、エラーを返す", func(t *testing.T) { /* ... */ })
    })
}
```

## 2. Naming

- All test case names are in **Japanese** and describe the behavior **and** the branch condition.
- The outer groups are the bare literals `正常系` / `異常系`. **Sub-case names inside them carry no `正常系_` / `異常系_` prefix** — they read as a behavior sentence (e.g. `firstNameの文字数が最小値未満の場合、エラーを返す`).

## 3. Parallelism

- Call `t.Parallel()` at **every nesting level**.
- The only exceptions are shared-mutable-state cases and env/CWD mutation (`t.Setenv`, `config.EnsureRepoRootAndEnv`), which are incompatible with `t.Parallel()`. Mark those with `//nolint:paralleltest` and a one-line reason. For a fixed listen port that collides under parallel packages, prefer an ephemeral address (`127.0.0.1:0`) over serializing the test.

## 4. No table-driven `for` loops

Write **sequential `t.Run` siblings, one per case** — do **not** use a `for _, tc := range cases` loop. Each case gets its own named `t.Run` so a failure names the exact scenario and parallelism is per-case.

## 5. Assertions

- Use `require` for **preconditions**, fatal checks, and **all error assertions** (`NoError` / `Error` / `ErrorIs` / `ErrorContains`). The `testifylint` `require-error` rule enforces this — `assert.ErrorIs` etc. fail lint.
- Use `require` for a check that **guards subsequent code** (e.g. `require.NotNil` before a dereference).
- Use `assert` for **terminal value verification** that does not guard later code (`Equal` / `Len` / `Contains` / `True` / `False` / `Empty`), so one run surfaces all mismatches.

```go
require.NoError(t, err)            // 前提（失敗で以降無意味）
require.ErrorIs(t, err, ErrX)      // エラー系は require（testifylint require-error）
assert.Equal(t, expected, actual)  // 終端の値検証は assert
```

## 6. Mocks and generated files

- Use the **generated mocks** under `*/mock/` (`go.uber.org/mock`). Never hand-write custom mocks in test files.
- Never edit generated files: `**/*.gen.go`, `*.sql.go`, `*_mock.go`.
- Tests rely only on public interfaces and generated artifacts.

## 7. Architectural rules in tests

Tests respect the same onion boundaries as production code:

- Domain tests must not use infrastructure implementations.
- Usecase tests mock domain repositories.
- Controller tests mock usecases.
- Do not bypass layers.

## 8. Coverage

- Run `make test` (coverage). Coverage **must not decrease** from the current baseline; new / modified packages exceed **90 %** and handlers approach ~100 %.
- If a package is below the bar, add the missing branch tests — do not stop until met. (The "done" definition lives in [`rules.md` → Testing & Definition of Done](rules.md).)

## 9. Coverage exceptions and governance

Some uncovered lines are legitimate and must **not** be chased with contrived tests:

- **Structurally unreachable** — an impossible `switch default`, a `panic` guarding a precondition that cannot fail, a compiler-mandated `return` after an exhaustive loop.
- **Infallible defensive branches** — error returns from operations that cannot fail in practice (e.g. `json.Marshal` of a `[]string`).
- **Write-once infrastructure (extraordinary measure / 超法規的措置)** — packages such as `internal/observability` that, once implemented, are rarely touched. Their defensive branches may be left uncovered **only when covering them would require extra production code, a signature change, or runtime-stack manipulation** — i.e. only branches reachable as-is are tested.

Rules for exceptions:

- Do **not** add contrived tests or extra implementation solely to color these lines.
- Record each exception in the owning package's `README` (the concrete file/function list).
- **Governance:** a new exception is **not added at will** — it is recorded only with an appropriate approver's (e.g. architect) sign-off. If an exempted function later gains real branching logic (not error plumbing), that logic must be unit-tested like everything else.
