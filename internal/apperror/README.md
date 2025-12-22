# app error

`apperror` パッケージは、層に依存しない「アプリケーション共通のエラー分類」を定義します。

## 基本方針

- Domain/Usecase/Controller/Infraのいずれからも参照可能。
- 定義するのは「基底カテゴリ」のみ。
  - 例: `ErrInvalidArgument`, `ErrNotFound` など。
- 具体的な HTTP ステータスコード変換やレスポンス整形は Controller 層で行う。

## 利用ルール

- エラー発生の起点では必ずこれら基底カテゴリをラップすることを推奨。
- `errors.Is` / `errors.As` を使ってController層でハンドリングを行っています。
- 新しいカテゴリを追加する場合は、この README に背景と利用シーンを明記すること。

## 対応表

| app error 定義 | 意味 / 使い所 | HTTP Status |
| -------------- | ----------- | ----------- |
| `ErrInvalidArgument` | 不正な引数（構文的には正しいが意味が不正） | 400 Bad Request |
| `ErrUnauthenticated` | 認証失敗（未ログインなど） | 401 Unauthorized |
| `ErrPermissionDenied` | 権限不足 | 403 Forbidden |
| `ErrNotFound` | 対象が存在しない | 404 Not Found |
| `ErrConflict` | 競合（ユニーク制約違反・同時更新衝突など） | 409 Conflict |
| `ErrValidation` | ドメイン/ユースケースの検証失敗 | 422 Unprocessable Entity |
| `ErrInternal` | 想定外の内部エラー | 500 Internal Server Error |
| `ErrUnimplemented` | 未実装 / 非サポート機能 | 501 Not Implemented |
| `ErrUnavailable` | 一時的な利用不可（外部依存障害など） | 503 Service Unavailable |
