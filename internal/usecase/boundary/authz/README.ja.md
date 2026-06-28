# authz

[English](README.md) | 日本語

認可（authz）のためのインターフェースと値オブジェクトを提供します。認証（`auth`）と対になる存在です。

## Authorizer

`Authorize(ctx, *auth.Authn, Action, *Resource) error` は、認証主体が `resource` に対して `action` を実行してよいかを判定します。許可する場合は `nil`、拒否する場合は `apperror.ErrPermissionDenied` をラップしたエラー（`ErrForbidden`、HTTP 403）を返します。

## 型

- `Action` — 認可対象の操作（例: `ActionUserDelete` = `"user:delete"`）。
- `Resource` — 対象リソース。`Kind()` と任意の `OwnerID()` を持ち、所有権ベース（オブジェクトレベル）の判定を表現できます。

## エラー

|エラー|説明|
|---|---|
|`ErrForbidden`|認可拒否（`apperror.ErrPermissionDenied` をラップ、HTTP 403）|

## 設計意図

- 認可判断を usecase 各所に散らさず、**差し替え可能なポリシー（PDP）** として表現する。
- `auth.Authn`（subject / scopes / claims）と対象 `Resource` を渡すことで、RBAC（claims からロール）と所有権（subject == OwnerID）の両モデルを表現可能にする。この `authz` boundary が兄弟の `auth` boundary に意図的に依存する（本層で唯一の boundary 間依存）のはこのためで、判断には id だけでなく認証主体全体が必要になる。

### 規約: usecase が呼出元をどう受け取るか

- リクエストで指定されたリソースに対する **オブジェクトレベル認可** を行う usecase メソッド（パスの `user_id` を対象にする `GetUser` / `UpdateUser` / `DeleteUser` 等）は、`*auth.Authn` 全体を受け取り先頭で `Authorizer.Authorize(...)` を呼ぶ。
- **呼出元自身の identity** のみをデータとして必要とするメソッド（認証ユーザー自身に作用する `CreateUser` / `ChangePassword` 等）は、controller 側で抽出したスカラ `uuid.UUID`（`authn.ID()`）を受け取る — 認可対象となる別オブジェクトが無いため。

## 実装

`internal/infrastructure/authz/` が `Authorizer` の実装を提供します。デフォルトの `allowall` 実装は全許可であり、本番以外の環境に限定されます。
