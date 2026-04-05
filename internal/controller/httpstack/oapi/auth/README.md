# auth (OpenAPI 認証連携)

概要: `auth` サブパッケージは、OpenAPI 検証パイプライン向けの認証関数を提供します。`NewAuthenticator` により `openapi3filter.AuthenticationFunc` を生成し、リクエストからトークンを抽出してアプリケーションの `Authenticator` に委譲し、認証情報（`Authn`）を `echo` コンテキストへ格納します。

## 役割

- OpenAPI の検証フェーズで呼ばれる認証関数を提供する (`NewAuthenticator`)。
- リクエストからのトークン抽出（Cookie および Header の順）を担う。
- 抽出したトークンを `authbd.Authenticator` に渡して認証を行い、成功した場合に `ctxhelper.SetAuthnToEcho` 経由で `Authn` を Echo コンテキストへセットする。

## 必要度

### 本番運用での必須度

- 必須度: 本番運用で推奨

理由: API レイヤーでの認証は本番環境で必須の機能です。本パッケージは OpenAPI ベースの検証と組み合わせて認証を行うため、仕様通りの認証フローを担保できます。

### 開発/テスト運用での必須度

- 必須度: 開発/テスト運用で推奨

理由: 認証ロジックを OpenAPI 検証パイプラインに組み込むことで、開発中やテスト時に認証まわりの不整合を早期に検出できます。ローカル用の簡易 `Authenticator` を DI で差し替えて利用することが可能です。

### 無効化した場合の影響

- 無効化すると、OpenAPI 検証フェーズで認証処理が行われず、ユースケース層での認可前提が崩れる可能性があります。認証情報がコンテキストにセットされないため、認証必須のエンドポイントは期待通りに動作しません。

## 注意点

- トークン抽出順序: 本実装はまず Cookie（`authCfg.CookieName()`）を確認し、続いて Header（`authCfg.HeaderName()`）を参照します。Header からの抽出では `authCfg.AllowedHeaderBearer()` が有効な場合に `Authorization: Bearer <token>` 形式を許容します。
- `NewAuthenticator` は `config.AuthConfig` と `authbd.Authenticator` に依存します。DI コンテナで適切にこれらを提供してください（例: `internal/di/module/auth.go`）。
- エラー群: パッケージは以下のエラー変数を定義しており、OpenAPI 検証パイプライン側でのハンドリングに使用できます:
  - `ErrInvalidAuthDefaultMode`：内部設定不整合
  - `ErrUnauthorizedTokenMissing`：トークンが見つからない（未認証）
  - `ErrUnauthorizedInvalidToken`：トークンが不正
  - `ErrUnauthorizedEchoContextNotFound`：Echo コンテキストがリクエストコンテキスト内に見つからない

- `authExtractor` はトークンが空の場合に nil を返す可能性があります。呼び出し側（OpenAPI 認証フック）がこの状態をどのように扱うかを確認してください。

## 使い方（簡易）

- OpenAPI ミドルウェア初期化時に `NewAuthenticator(authCfg, authenticator)` を渡して認証フックを登録します。
- DI により `authCfg`（`config.AuthConfig`）と `authenticator`（`authbd.Authenticator`）を注入してください。
