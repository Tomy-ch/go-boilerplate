# このプロジェクトが意図的に含めていないもの

## 企業のインフラ選択に依存するもの

- デプロイ実装
  - スケルトンのみ提供：[.github/workflows/deploy-app.yaml](../../.github/workflows/deploy-app.yaml)
- IaC 実装
- Observability 運用設定
- サーキットブレーカー
- シークレットローテーション
- レートリミット
  - アプリ内（インメモリ）リミッターとしては意図的に提供しない
  - クラウドネイティブなマルチインスタンス構成では、インスタンスごとの
    インメモリカウンタは状態を共有できず、正しいグローバル制限を担保できない
  - これはインフラのエッジ（API ゲートウェイ / ロードバランサー /
    リバースプロキシ / サービスメッシュ）の責務
- スケジュールジョブの多重実行制御
  - スケジュールジョブの重複 / マルチインスタンスの防護（k8s CronJob の
    `concurrencyPolicy`、advisory lock）はスケジューラ側へ委ねる
  - 同梱ジョブはいずれも設計上すでに並行安全なので、アプリケーション層の
    排他は提供しない: `outbox-gc` と `idempotency-gc` は経過時間述語による
    べき等なバッチ削除、`usercount` は読み取り専用、outbox relay は
    `FOR UPDATE SKIP LOCKED` で行を確保する
  - 厳密な単一実行が要るなら、スケジューラ側で
    `concurrencyPolicy: Forbid` を設定する

## ドメイン要件に強く依存するもの

- 監査ログ
- RBAC / 認可モデル
- セッション管理
- パスワードポリシー
  - リポジトリ内のクレデンシャルストアは提供しない：認証は外部の OIDC / JWT (Bearer) IdP に委譲されるため、本サービスはパスワードを一切保持しない。[docs/design/auth.md](../design/auth.ja.md) を参照。
- データ保持ポリシー
  - 論理削除はサンプルで提供
- PII保存時の暗号化

## 利用者が独自に実装することを想定しているもの

- 認証形式 （JWT, Cookie, OAuth2 など）
  - サンプル実装は提供。拡張可能な設計を採用。
    - インターフェイス：[internal/usecase/boundary/auth/authenticator.go](../../internal/usecase/boundary/auth/authenticator.go)
    - ローカル・テスト用：[internal/infrastructure/auth/local/auth_local.go](../../internal/infrastructure/auth/local/auth_local.go)
- アカウントロックアウト
- データエクスポート / 削除権対応
- キャッシュ層
  - 専用のキャッシュ抽象は、検討した上で意図的に棄却した
  - 汎用的な `Cache` インターフェイスは最小公倍数（TTL 付き map）に退化し、
    実装のセマンティクスが漏れる上に、技術固有の機能（Redis の pipeline /
    Lua / pub-sub など）を捨ててしまう
  - キャッシュが必要な場合は、既存の domain Repository インターフェイスを
    満たすデコレータとして実装する。これにより domain / usecase はキャッシュの
    存在を知らないまま済む。Repository インターフェイスが差し替えの seam を
    既に提供しているため、新たな抽象は不要
