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
- `auth.Authn`（subject / scopes / claims）と対象 `Resource` を渡すことで、RBAC（claims からロール）と所有権（subject == OwnerID）の両モデルを表現可能にする。

## 実装

`internal/infrastructure/authz/` が `Authorizer` の実装を提供します。デフォルトの `allowall` 実装は全許可であり、本番以外の環境に限定されます。
