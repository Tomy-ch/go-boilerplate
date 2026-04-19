# migrations

[English](README.md) | 日本語

`database/migrations` は、**golang-migrate によるデータベースマイグレーションファイル**を格納するディレクトリです。

## ファイル生成

新しいマイグレーションファイルは以下のコマンドで生成します。

```bash
make new-migrate-<name>
```

例：

```bash
make new-migrate-create_orders
```

これにより、連番付きの up / down ペアが自動生成されます。

```text
000010_create_orders.up.sql
000010_create_orders.down.sql
```

## ファイル命名規則

```text
{6桁連番}_{説明}.up.sql    # アップグレード（適用）
{6桁連番}_{説明}.down.sql  # ダウングレード（ロールバック）
```

- 連番は `000001` から始まる6桁ゼロ埋め
- 説明はスネークケースで内容を簡潔に表す
- up と down は必ずペアで作成する

## マイグレーション実行

|コマンド|説明|
|---|---|
|`make migrate-up`|すべての未適用マイグレーションを適用|
|`make migrate-down`|直前のマイグレーションをロールバック|

CLI からも実行可能です。

```bash
./server migrate-up
./server migrate-down
```

## ルール

- **既存のマイグレーションファイルを変更しない** — 適用済みのファイルを変更するとハッシュ不整合が発生する
- **必ず新しいファイルを作成する** — スキーマ変更は常に新規マイグレーションで行う
- **up と down はペアで作成する** — down がないとロールバックできない
- **down は up の逆操作を正確に記述する** — テーブル作成なら DROP、カラム追加なら DROP COLUMN
- **冪等性を意識する** — `IF NOT EXISTS` / `IF EXISTS` を活用する
- **連番に欠番を作らない** — CI でギャップ検出される（`migration-check.yaml`）

## CI チェック

`migration-check.yaml` ワークフローにより、以下が自動検証されます。

- 連番の重複がないこと
- 連番に欠番がないこと
- up / down のペアが揃っていること

## 注意点

- このディレクトリには DDL（テーブル定義・スキーマ変更）のみ配置する
- DML（データ操作）は `database/dml/` に配置する
- シードデータは `database/seed/` に配置する
- 初期化用 SQL（DB 作成・拡張有効化）は `docker/database/sql/` に配置する
