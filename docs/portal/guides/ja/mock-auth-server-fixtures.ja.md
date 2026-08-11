# mock-auth-server fixtures

[English](README.md) | 日本語

> このファイルは canonical な [README.md](README.md) の日本語訳です。内容の更新は canonical 側で行い、本ファイルへ同期してください。

疑似 OIDC provider 用の固定・例示ユーザー。

## `users.json`

例示ユーザーの配列。用途は次の3つのみ:

- `GET /test/users`（一覧。将来の Login UI 用）
- `POST /test/token` を `subject` 省略で呼んだ際のフォールバック
- 起動ログの件数

`POST /test/token` は渡した `subject` を**そのまま**トークンにするため、このファイルが
無い/空でも mock は動作する（users は `[]` 扱い）。

### スキーマ

| フィールド | 型 | 説明 |
| --- | --- | --- |
| `subject` | string | OIDC `sub`（外部 identity の識別子） |
| `email` | string | メールアドレス |
| `given_name` | string | 名 |
| `family_name` | string | 姓 |
| `name` | string | 表示名 |
| `status` | string | `active` / `deleted` / `unregistered`（未使用） |

### ユーザーの登録

配列にエントリを追加する:

```json
{
  "subject": "user-example",
  "email": "user@example.com",
  "given_name": "Example",
  "family_name": "User",
  "name": "Example User",
  "status": "active"
}
```

## Reset

`users.json` を中立な既定内容（汎用ユーザー 1 件）で（再）生成するには:

```sh
make reset-mock-auth-users-ci
# または tool runner 経由:
make reset-mock-auth-users
```
