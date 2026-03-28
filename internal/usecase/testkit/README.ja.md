# `testkit` パッケージ

概要: ユースケース層のテストを支援するヘルパーを提供するパッケージです。本ディレクトリには、ユースケースの単体テストで頻繁に必要となるテスト用エラー生成やモック化されたトランザクションマネージャーのユーティリティが含まれます。

## 提供する主な機能

- `ExpectedDBError(t *testing.T) error`
  - テストで「DBエラーを表す固定のエラー」を簡単に生成するためのヘルパーです。テストの期待値として使います。

- `NewMockTransactionManager(t *testing.T) tx.Manager`
  - gomock を用いて `tx.Manager` のモックを生成します。返却されるモックは `Do(ctx, fn)` を呼ばれた際に `fn(ctx)` をそのまま実行する振る舞いを持ち、`.AnyTimes()` により何度でも使えるよう設定されています。ユースケースのトランザクションまわりのロジックをテストする際に便利です。

## 使い方（例）

1) DBエラーの期待値を作る

    ```go
    func Test_SomeUsecase_DBError(t *testing.T) {
      expected := usecasetest.ExpectedDBError(t)
      // expected をモックの返却値や期待エラー判定に利用
    }
    ```

2) トランザクションマネージャーのモックを使う

    ```go
    func Test_SomeUsecase_WithTx(t *testing.T) {
      mockTx := usecasetest.NewMockTransactionManager(t)
      // mockTx をユースケースの依存に注入してテストを実行
    }
    ```

このモックは `Do(ctx, fn)` が呼ばれると `fn(ctx)` をその場で実行するため、トランザクション内の処理が呼ばれることを確認できます。

## 注意点

- `NewMockTransactionManager` は内部で gomock を利用します。テスト内で生成したモックコントローラはテスト終了時に自動的に検証されます。
- 本パッケージのユーティリティはテスト補助のためのものです。本番コードに組み込むべきではありません。

## テスト

- 本パッケージ自体にも単体テストが含まれている場合があります。`go test ./internal/usecase/usecasetest` で実行できます。
