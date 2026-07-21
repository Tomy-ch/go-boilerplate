# mock-auth-server（JWT Test Provider）

[English](README.md) | 日本語

> このファイルは canonical な [README.md](README.md) の日本語訳です。内容の更新は canonical 側で行い、本ファイルへ同期してください。

ローカル開発用の**疑似 OIDC 認証サーバー**。実 IdP なしに、Go API（Resource Server）が **JWT の署名・Claim 検証**を E2E で検証できるようにする。スコープは**署名 + Claim 検証まで**。`iss+sub → User` 解決・Role・Authorization Code Flow は**未実装**。

Token 発行側と検証側が同一ライブラリ由来の誤りを共有しないよう、**Go API とは異なるランタイム（Node.js 上の TypeScript）**で実装している。`.ts` は Node 24 のネイティブ型ストリッピングで直接実行するため、`tsc` ビルドも追加ランタイム依存も不要（`typescript` は `npm run typecheck` 用の dev 依存のみ）。

## エンドポイント

| Method | Path | 説明 |
| --- | --- | --- |
| GET | `/health` | 死活確認 — `{"status":"ok"}` |
| GET | `/.well-known/openid-configuration` | OIDC Discovery ドキュメント |
| GET | `/.well-known/jwks.json` | JWKS（公開鍵のみ: `kty=RSA` `alg=RS256` `use=sig` `kid=mock-key-1`） |
| POST | `/bypass/token` | Token 発行 — `{subject, profile}` → `{access_token, token_type, expires_in}` |
| GET | `/admin/users` | 固定 User Fixture の一覧 |

`/bypass/*` ・ `/admin/*` は**テスト / dev 専用**。`MOCK_AUTH_DEV_ENDPOINTS=disabled` で本番相当モードでは `404` を返す。OIDC の全表面（`/oidc/authorize` ・ `/oidc/token` ・ `/oidc/userinfo` ・ `/oidc/logout`）は `openapi/` に定義済みで、後続の Increment で実装する。

## Token Profile（`/bypass/token`）

異常系は再現性のため**固定 Profile 方式**（任意 Claim 注入 API にはしない）。Go API は `valid` を受理し、各異常系を `401` で拒否することが期待値。

| Profile | 内容 | Go API |
| --- | --- | --- |
| `valid` | 正常な access token（`typ=at+jwt`・正しい `iss`/`aud`/`exp`/`nbf`/`sub`） | 受理 |
| `expired` | `exp` が過去 | 401 |
| `not-yet-valid` | `nbf` が未来 | 401 |
| `wrong-issuer` | `iss` 不一致 | 401 |
| `wrong-audience` | `aud` 不一致 | 401 |
| `missing-subject` | `sub` 無し | 401 |
| `invalid-signature` | JWKS に無い鍵で署名（`kid` は正規） | 401 |
| `unsupported-algorithm` | `HS256`（対称・RS256 allowlist 外） | 401 |
| `id-token` | `token_use=id`・`aud=<client_id>`・`typ` が `at+jwt` でない | 401 |

valid の access token クレームはデファクト標準プロファイルに従う: `iss` / `sub` / `aud` / `exp` / `iat` / `nbf` / `jti` / `client_id` / `token_use=access` / `scope`。access token は `typ=at+jwt` ヘッダ（RFC 9068）を持ち、Go 側検証はこの typ で ID Token 誤用を拒否する。

## 鍵と Fixture

- `keys/mock-key-1.pem` — **固定 RSA 秘密鍵**（再起動で不変 = 発行 Token の再現性）。`kid=mock-key-1`。
- `fixtures/users.json` — 例示 User（Subject / Email / 名前。`status` は未使用）。ファイルが無くても mock は動作する（`/bypass/token` は任意 `subject` を受理）。詳細は `fixtures/README.md`。

## 使い方

```sh
docker compose --profile development up mock_auth_server
curl -s http://localhost:4000/.well-known/openid-configuration
curl -s -X POST http://localhost:4000/bypass/token \
  -H 'Content-Type: application/json' \
  -d '{"subject":"user-john-doe","profile":"valid"}'
```

Go API は `AUTH_ISSUER` にホスト URL（`http://localhost:4000`）、`AUTH_JWKS_URL` に内部サービス URL（`http://mock_auth_server:4000/.well-known/jwks.json`）を指定する。**issuer と JWKS 取得 URL を分離**し、`iss` はブラウザ / ホストから解決可能な URL のまま、鍵取得はコンテナ名で行う。詳細は `env/README.md`（Auth セクション）。

## 環境変数

| 変数 | 既定値 | 説明 |
| --- | --- | --- |
| `OIDC_PORT` | `4000` | 待ち受けポート |
| `OIDC_ISSUER` | `http://localhost:4000` | `iss` クレーム / OIDC issuer（ホストから解決可能な URL） |
| `OIDC_AUDIENCE` | `go-boilerplate-api` | access token の `aud` クレーム |
| `OIDC_CLIENT_ID` | `go-boilerplate-client` | `client_id` クレーム / ID Token の audience |
| `MOCK_AUTH_DEV_ENDPOINTS` | `enabled` | `disabled` で `/bypass/*` ・ `/admin/*` を `404` にする |

## 未実装

`user_identities` / Role / deleted・unknown user、複数 `kid` による Key Rotation（`/admin/keys/rotate`）、Authorization Code Flow / Login UI / `/oidc/userinfo` / `/oidc/logout`。
