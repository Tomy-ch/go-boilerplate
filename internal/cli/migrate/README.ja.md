# migrate-up / migrate-down

[English](README.md) | 日本語

データベーススキーマのマイグレーションを管理します。`migrate-up` は未適用のマイグレーションを適用し、`migrate-down` は適用済みのマイグレーションをロールバックします（既定では全件、または指定した段数）。

## コマンド

```text
migrate-up
migrate-down
```

## フラグ

|フラグ|デフォルト|説明|
|---|---|---|
|`--steps`|`0`|現在位置からの相対的な適用段数。`0` で全件、正の整数でその段数だけ適用します。負値は拒否されます。|
|`--database`|`""`|対象データベース名（例: `local`, `test`）。空の場合は設定値の既定 DB を使用します。|

`--steps` は**相対的な段数**であり、絶対的なターゲットバージョンではありません。例えば `migrate-up --steps 2` は現在位置から 2 段進め、`migrate-down --steps 2` は 2 段巻き戻します。

## 使い方

```bash
# 未適用のマイグレーションをすべて適用
./server migrate-up

# 次の 2 段だけ適用
./server migrate-up --steps 2

# すべてのマイグレーションをロールバック
./server migrate-down

# 直近 2 段をロールバック
./server migrate-down --steps 2

# 対象データベースを指定
./server migrate-up --database test
```

## 注意点

- マイグレーションファイルは `database/migrations` に配置されています。
- **本番環境での `migrate-down` は慎重に実行してください。** データ損失のリスクがあるため、必ず事前にバックアップを取得してください。
- 既存のマイグレーションファイルは変更せず、常に新しいファイルを作成してください。
- `--steps` を指定しない全件 `migrate-down` は、ロールバック前に `dirty` 状態のデータベースを現在バージョンで整合させるため、過去に失敗したマイグレーションがロールバックを妨げません。
- `ErrNoChange`（既に対象位置にある場合）は成功として扱われます。
