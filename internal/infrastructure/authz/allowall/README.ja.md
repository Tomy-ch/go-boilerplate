# allowall（ローカル開発用の全許可 Authorizer）

[English](README.md) | 日本語

**すべての要求を許可する**単純な `Authorizer` 実装です。ローカル開発・CI / テスト環境専用であり、**本番利用は想定していません**。

## 役割

- 実際の認可ポリシーが存在しない段階でも機能開発を進められるよう、`Authorizer` 依存を満たす。
- `Authorize(...)` は subject / action / resource によらず常に `nil`（許可）を返す。

## fail-closed な生成

`New` は `*config.ApplicationConfig` を受け取り、**`local` / `ci` / `test` 以外では生成を拒否**して本番相当の環境ではエラーを返します。すべてを許可するスタブは危険なため、この前提条件は呼び出し側ではなくスタブ自身が担保します —— `provideAuthorizer` の配線を誤っても、本番で全許可ポリシーに到達することはありません。DI プロバイダはその拒否を起動失敗として表面化させます。

## 本番向けの差し替え

`allowall` を RBAC / 外部ポリシーエンジン（OPA / Cedar）実装に差し替え、本番相当の環境向けに `provideAuthorizer`（`internal/di/module/authz.go`）で配線してください。

## 注意点

- 認可を一切行いません（全許可）。
- 本番で使用しないこと。
