# `testkit` Package

English | [日本語](README.ja.md)

Overview: This package provides helpers to support testing in the Usecase layer. This directory contains utilities such as test error generation and mocked transaction managers that are frequently required in Usecase unit tests.

## Main Features Provided

- `ExpectedDBError(t *testing.T) error`
  - A helper to easily generate a "fixed error representing a DB error" in tests. It is used as an expected value in tests.

- `NewMockTransactionManager(t *testing.T) tx.Manager`
  - Generates a mock of `tx.Manager` using gomock. The returned mock has the behavior of executing `fn(ctx)` as-is when `Do(ctx, fn)` is called, and is configured with `.AnyTimes()` so it can be used any number of times. It is useful when testing transaction-related logic in Usecase.

## Usage (Example)

1) Create an expected DB error

    ```go
    func Test_SomeUsecase_DBError(t *testing.T) {
      expected := testkit.ExpectedDBError(t)
      // Use expected as the return value of mocks or for expected error assertions
    }
    ```

2) Use a mock transaction manager

    ```go
    func Test_SomeUsecase_WithTx(t *testing.T) {
      mockTx := testkit.NewMockTransactionManager(t)
      // Inject mockTx into Usecase dependencies and execute the test
    }
    ```

This mock executes `fn(ctx)` immediately when `Do(ctx, fn)` is called, allowing you to verify that processing inside the transaction is invoked.

## Notes

- `NewMockTransactionManager` internally uses gomock. The mock controller created in the test is automatically verified at the end of the test.
- The utilities in this package are for test support purposes. They must not be included in production code.

## Test

- This package itself may include unit tests. You can run them with `go test ./internal/usecase/testkit`.
