# server

概要:
`server` ディレクトリは、アプリケーションの **HTTP サーバー本体（Echo）を初期化し、DI ライフサイクルに統合するためのコントローラ層** です。
`NewAppServer` による Echo インスタンス生成、および `ServeHTTP` による起動・停止登録を担い、
extension 配下で適用されたミドルウェア・サーバー設定を最終的に反映した **完成された HTTP サーバー** を起動します。

## 役割

- Echo インスタンスの生成 (`NewAppServer`)
- DI ライフサイクル（fx / lifecycle.Registrar）へサーバー起動処理を登録
- サーバー起動 (`Start`) と停止 (`Shutdown`) の制御関数を提供
- ServerConfig / ApplicationConfig / SecurityConfig に基づくサーバー実行設定の適用
- extension（middleware, security, logging, observability など）が反映された完全なサーバーを起動する

このディレクトリは **「最終的な HTTP サーバーの起動ポイント」** の責務を持ちます。

## 必要度

### 本番運用での必須度

- 必須度: **本番運用で必須**

理由:

- HTTP サーバーが起動しなければ API が提供されないため
- 拡張済みミドルウェア（security, cors, logging, observability）を適切な順序で適用し起動する役割を持つため
- 安全な shutdown（graceful shutdown）を保証し、接続中のクライアントへの影響を最小化するため
- 起動ログ（port, allowed_origins, CIDR, mode）が本番運用の可観測性として重要

### 開発/テスト運用での必須度

- 必須度: **開発/テスト運用で必須**

理由:

- ローカルでの API 開発には Echo サーバーが必須
- integration / E2E テストで HTTP サーバーの起動が必要
- 開発/テスト環境でもミドルウェアが正しく適用されているか確認する必要がある
- shutdown のテスト（timeouts や graceful）も実際に利用するため

## 無効化した場合の影響

- HTTP サーバーが起動せず、どの API も動作しない
- DI によるサーバー制御が機能せず、アプリ全体が破綻
- ミドルウェア適用後の Echo を使用できず、セキュリティ / ログ / バリデーションが無効化される
- graceful shutdown が動作せず、接続切断や不整合が生じる可能性

**→ このディレクトリは API アプリケーションの稼働に不可欠です。**

## 注意点

- `ServeHTTP` は **fx / lifecycle.Registrar に Start / Shutdown を登録するだけ**
  → 直接呼び出してはいけない（DI によるライフサイクル管理が壊れる）
- `NewAppServer` で生成した Echo インスタンスには、後続の extension（middleware / configurator）が適用される
  → このディレクトリでは **middleware を直接定義しない**
- ログには logging.Logger を使用し、zap の生利用は禁止（封印層のため）
- セキュリティ情報（allowed_origins / CIDR）は起動ログで確認可能
- graceful shutdown の Timeout は ServerConfig に従うため、設定ミスに注意すること

以上が `server` ディレクトリの README です。
