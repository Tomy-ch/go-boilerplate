# このプロジェクトが意図的に含めていないもの

## 企業のインフラ選択に依存するもの

- デプロイ実装
  - スケルトンのみ提供：[.github/workflows/deploy-app.yaml](.github/workflows/deploy-app.yaml)
- IaC 実装
- Observability 運用設定
- サーキットブレーカー
- シークレットローテーション

## ドメイン要件に強く依存するもの

- 監査ログ
- RBAC / 認可モデル
- セッション管理
- パスワードポリシー
  - サンプル実装は提供。拡張可能な設計を採用。
    - インターフェイス； [internal/usecase/boundary/security/encrypt_hasher.go](internal/usecase/boundary/security/encrypt_hasher.go)
    - サンプル実装； [internal/infrastructure/security/bcrypt_hasher.go](internal/infrastructure/security/bcrypt_hasher.go)
- データ保持ポリシー
  - 論理削除はサンプルで提供
- PII保存時の暗号化

## 利用者が独自に実装することを想定しているもの

- 認証形式 （JWT, Cookie, OAuth2 など）
  - サンプル実装は提供。拡張可能な設計を採用。
    - インターフェイス：[internal/usecase/boundary/auth/authenticator.go](internal/usecase/boundary/auth/authenticator.go)
    - ローカル・テスト用；[internal/infrastructure/auth/local/auth_local.go](internal/infrastructure/auth/local/auth_local.go)
- アカウントロックアウト
- データエクスポート / 削除権対応
