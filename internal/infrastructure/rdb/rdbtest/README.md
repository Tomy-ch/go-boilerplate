# `rdbtest` パッケージ

概要: テスト用の RDB 接続インスタンスやテスト向けトランザクション管理を提供するユーティリティ群です。リポジトリのユニット/統合テストで、実装済みインフラや New 関数の振る舞いを検証するときに使います。

## 目的

- テスト用 DB インスタンスとログ付き DB プロバイダを簡単に作成するためのヘルパーを提供します。
- テスト実行中の DB 操作をロールバックするトランザクションヘルパーを提供し、テストの独立性を保ちます。

## 主要 API（実装に合わせた説明）

- `NewTestDBWithLoggingProvider(t *testing.T) (driver.DatabaseDriver, loggingdb.DBProvider)`
  - テスト用の `driver.DatabaseDriver`（実際の DB へ接続）と、ログ付きの `loggingdb.DBProvider` を生成します。テスト用の設定を内部で読み込みます。

- `NewTestTransactionManager(t *testing.T) TransactionRunner`
  - トランザクション内で処理を実行し、終了時に必ずロールバックするためのテスト用トランザクションマネージャーを返します。

- `TransactionRunner`（インターフェース）
  - `WithinTx(fn func(ctx context.Context))` を実装します。渡した関数をトランザクション内で実行し、終了時にロールバックします。

実装上の補足:

- テスト内で行った操作を DB に残さないため、`WithinTx` は内部で意図的にエラー（`rollbackForTestError`）を返してロールバック処理を誘発し、エラーがその特殊値の場合は成功扱いとして処理します。

## 使用例

例: トランザクション内でリポジトリの操作を行い、変更を残さずに検証する。

```go
txm := rdbtest.NewTestTransactionManager(t)
txm.WithinTx(func(ctx context.Context) {
    // ここで tx コンテキストを使ってリポジトリを操作する
    // 例: repo := repository.NewRepo(db)
    // repo.Create(ctx, ...)
    // テスト終了時に必ずロールバックされる
})
```

例: ログ付き DB プロバイダを取得してインフラの New() に渡す

```go
db, loggingProvider := rdbtest.NewTestDBWithLoggingProvider(t)
// db/loggingProvider を使って実際のリポジトリ/インフラを初期化
```

## テスト

- 本ディレクトリには単体テストが含まれており、テストインスタンス生成や `WithinTx` のロールバック振る舞いが検証されています。`go test ./...` で実行できます。

## 注意点

- テスト用インスタンスは実際に DB に接続します。CI やローカルで実行する際はテスト用 DB の接続情報が正しく設定されていることを確認してください。
- `WithinTx` は常にロールバックを行う動作です。永続的な副作用を検証したい場合は別途専用のセットアップを使ってください。
