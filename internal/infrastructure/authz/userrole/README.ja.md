# userrole（user_roles ベースの Authorizer）

[English](README.md) | 日本語

`user_roles` テーブルに基づく `Authorizer` 実装です。`allowall` のサンプル対となる実装で、本番相当環境に配線され、fail-closed エラーの代わりに実際の（ロールベースの）認可ポリシーで起動できるようにします。

## 役割

- 認証主体に割り当てられたロールを用いて `Authorizer` boundary（`internal/usecase/boundary/authz`）を満たす。
- ロール所属とリソース所有権から可否を判定する。

## ポリシー

`Authorize(ctx, authn, action, resource)` は次の順で判定します:

1. 内部 UserID（`authn.UserID()`）が解決済みかつゼロ値でないこと。未解決またはゼロ値なら拒否。ゼロ値 UUID は解決済みの主体を指さず、手順 4 は値の一致だけを見るため、通すとゼロ値の主体がゼロ値の所有者と一致してしまう。
2. `user.RoleRepository` で主体のロールを取得する。
3. 管理者ロール（`RoleCodeAdmin`）を持つ場合は許可。
4. それ以外は、主体がリソース所有者本人（`subject == Resource.OwnerID()`）の場合のみ許可。
5. いずれにも該当しなければ拒否し、`ErrForbidden`（`apperror.ErrPermissionDenied`、HTTP 403 を wraps）を返す。

個別 API の action 別認可は各 usecase で行います。本実装はロール/所有権によるベースライン判定を提供します。

## DI

`provideAuthorizer`（`internal/di/module/authz.go`）の `default`（本番相当）環境に配線します。`local` / `ci` / `test` は `allowall` を維持します。

## 注意点

- サンプルドメインの一部です。`make setup-remove-sample-api` で `user` サンプルと共に削除され、削除後は `provideAuthorizer` が fail-closed エラーへ戻ります。
