# allowall（ローカル開発用の全許可 Authorizer）

[English](README.md) | 日本語

**すべての要求を許可する**単純な `Authorizer` 実装です。ローカル開発・CI / テスト環境専用であり、**本番利用は想定していません**。

## 役割

- 実際の認可ポリシーが存在しない段階でも機能開発を進められるよう、`Authorizer` 依存を満たす。
- `Authorize(...)` は subject / action / resource によらず常に `nil`（許可）を返す。

## 本番向けの差し替え

DI プロバイダ（`internal/di/module/authz.go` の `provideAuthorizer`）は環境ゲート付きです。`allowall` は local / CI / test のみに配線し、本番相当の環境ではエラーを返すため、全許可ポリシーが本番に出荷されることはありません。RBAC / 外部ポリシーエンジン（OPA / Cedar）実装に差し替えてください。

## 注意点

- 認可を一切行いません（全許可）。
- 本番で使用しないこと。
