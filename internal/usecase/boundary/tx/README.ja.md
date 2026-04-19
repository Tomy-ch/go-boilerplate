# tx

[English](README.md) | 日本語

トランザクション境界管理のための `Manager` インターフェースと、トランザクション内で値を返すジェネリクスヘルパーを提供します。

## 公開 API

|型 / 関数|説明|
|---|---|
|`Manager`|`Do(ctx, fn)` — `fn` をトランザクション内で実行（成功時 commit、エラー時 rollback）|
|`DoWithResult[T](ctx, m, fn)`|トランザクション内で値を返すジェネリクスヘルパー|

## 設計意図

- Usecase に「トランザクションの存在」を意識させつつ、DB の詳細は隠蔽
- DB ドライバ依存（pgx, sql.Tx 等）を完全に隠蔽
- トランザクション境界は Usecase の責務 — Infrastructure はトランザクションを開始しない

## 実装

`internal/infrastructure/rdb/driver/` に pgx トランザクションを使った具体実装が配置されています。

## 注意点

- `Manager.Do` はネスト呼び出しに対応 — context に既存トランザクションがあれば再利用
- `DoWithResult` は `Manager.Do` をラップして型付き戻り値を抽出
- トランザクションスコープは不要なロックを避けるため最小限に保つこと
