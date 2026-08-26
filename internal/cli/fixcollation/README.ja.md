# fix-collation

[English](README.md) | 日本語

PostgreSQL の照合順序バージョン不整合を、`REINDEX DATABASE` と `ALTER DATABASE ... REFRESH COLLATION VERSION` を実行して修正します。

## 役割

このコマンドは照合順序（collation）のずれから復旧するために存在します。稼働中のデータベースの下で OS の照合ライブラリがアップグレードされると、既存のテキストインデックスのソート順がデータベースの前提とする順序と知らぬ間に一致しなくなり、インデックスを再構築するまでクエリ結果が誤るおそれがあります。本コマンドは該当インデックスを再構築し、記録された照合順序バージョンを再スタンプすることで、警告を解消しインデックスを再び信頼できる状態に戻します。再構築と再スタンプの判定ロジックを薄いコマンド配線から切り離して単体検証できるよう、純粋かつテスト可能なコアとして独立しています。

## コマンド

```text
fix-collation [flags]
```

## フラグ

|フラグ|デフォルト|説明|
|---|---|---|
|`--database`|`local`|対象データベース名。`local` / `test` / `template1` / `wt<N>_local` / `wt<N>_test` のいずれか|

## 使い方

```bash
./server fix-collation --database local
./server fix-collation --database template1   # CREATE DATABASE ... TEMPLATE template1 の失敗を解消する
./server fix-collation --database wt3_test    # worktree スロットが借りているデータベース
```

## 注意点

- **開発・テスト用データベースでの使用を想定しています。** 上記以外の名前は SQL 実行前にエラーで拒否されます。データベース名は SQL 文へ文字列として埋め込むため、この許可リストはインジェクション対策も兼ねています。
- `template1` を対象に含めているのは、不整合がそこから複製される全データベースへ伝播するためです。古い照合順序バージョンを抱えたままだと `CREATE DATABASE ... TEMPLATE template1` 自体が失敗します。
- `wt<N>_local` / `wt<N>_test` は worktree がスロットプールから借りるデータベースです（`docs/maintenance/db-worktree-pool.md` 参照）。
- **修正対象のデータベースへ接続し直します**（設定の DSN のデータベース名を上書きします）。`REINDEX DATABASE` は接続中のデータベースしか対象にできないため、設定の接続先のままでは他の名前を指定しても失敗します。
- `psql` が `PATH` 上に存在し、実行ユーザーが `REINDEX` および `ALTER DATABASE` の権限を持っている必要があります。
- データベース名以外の接続情報（ホスト・ポート・ユーザー・`sslmode`）はアプリケーション設定（`DSN`）から取得されます。
- SQL 実行時にエラーが発生すると即座に停止します（`ON_ERROR_STOP=1`）。
