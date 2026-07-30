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
4. それ以外は、主体がリソース所有者本人（`subject == Resource.OwnerID()`）の場合のみ許可。`OwnerID()` が `nil` のリソースは比較対象となる所有者を持たないため、このフォールバックは決して成立せず、許可し得るのは手順 3 のみ —— 実質的に **admin 限定** として振る舞う。したがって所有者を持たせずに `Resource` を組み立てることが、呼び出し側が admin 限定操作を宣言する方法であり、安全側に倒れる。所有者の指定漏れはアクセスを狭めるだけで、広げることはない。
5. いずれにも該当しなければ拒否し、`ErrForbidden`（`apperror.ErrPermissionDenied`、HTTP 403 を wraps）を返す。

個別 API の action 別認可は各 usecase で行います。本実装はロール/所有権によるベースライン判定を提供します。

## DI

`provideAuthorizer`（`internal/di/module/authz.go`）で配線します。`user` サンプルが存在する間は `ci` / `test` が `allowall` を受け取り、それ以外の既知の環境 —— `local` / `development` / `staging` / `production` —— は本実装を受け取ります。したがってローカル実行でも実際のロールベース判定が働きます。サンプルを削除すると `local` は `allowall` の case に畳み込まれ、本番相当環境は fail-closed エラーへ戻ります。

## 注意点

- サンプルドメインの一部です。`make setup-remove-sample-api` で `user` サンプルと共に削除され、削除後は `provideAuthorizer` が fail-closed エラーへ戻ります。
