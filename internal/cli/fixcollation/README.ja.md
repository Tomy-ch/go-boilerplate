# fix-collation

[English](README.md) | 日本語

PostgreSQL の照合順序バージョン不整合を、`REINDEX DATABASE` と `ALTER DATABASE ... REFRESH COLLATION VERSION` を実行して修正します。

## コマンド

```text
fix-collation [flags]
```

## フラグ

|フラグ|デフォルト|説明|
|---|---|---|
|`--database`|`local`|対象データベース名|

## 使い方

```bash
./server fix-collation --database local
```

## 注意点

- **ローカルおよびテスト用データベースでの使用を想定しています。** 本番環境で実行する場合はメンテナンス時間帯に行ってください。
- `psql` が `PATH` 上に存在し、実行ユーザーが `REINDEX` および `ALTER DATABASE` の権限を持っている必要があります。
- データベース接続情報はアプリケーション設定（`DSN`）から取得されます。
- SQL 実行時にエラーが発生すると即座に停止します（`ON_ERROR_STOP=1`）。
