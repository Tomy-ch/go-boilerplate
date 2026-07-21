# mock-auth-server（JWT Test Provider）

[English](README.md) | 日本語

> このファイルは canonical な [README.md](README.md) の日本語訳です。内容の更新は canonical 側で行い、本ファイルへ同期してください。

ローカル開発用の**疑似 OIDC 認証サーバー**。実 IdP なしに、Go API（Resource Server）が **JWT の署名・Claim 検証と Authorization Code Flow（+ PKCE S256）**を E2E で検証できるようにする。Login UI / consent / refresh token / Role は対象外。

Token 発行側と検証側が同一ライブラリ由来の誤りを共有しないよう、**Go API とは異なるランタイム（Node.js 上の TypeScript）**で実装している。`.ts` は Node 24 のネイティブ型ストリッピングで直接実行するため `tsc` ビルドは不要で、HTTP 層は [Hono](https://hono.dev/)。`typescript` は `npm run typecheck` 用の dev 依存で、`node --test` でテストを実行する。

## エンドポイント

| Method | Path | 説明 |
| --- | --- | --- |
| GET | `/health` | 死活確認 — `{"status":"ok"}` |
| GET | `/.well-known/openid-configuration` | OIDC Discovery ドキュメント |
| GET | `/.well-known/jwks.json` | JWKS（公開鍵のみ: `kty=RSA` `alg=RS256` `use=sig` `kid=mock-key-1`） |
| GET | `/oidc/authorize` | 認可エンドポイント（Code Flow + PKCE S256）— `302` で `code`、または error `redirect` / `400` |
| POST | `/oidc/token` | トークンエンドポイント（`authorization_code` + PKCE）— `access_token` + `id_token` を発行 |
| GET | `/oidc/userinfo` | UserInfo（Bearer access token・whitelist claim）。ID Token は `401` で拒否 |
| POST | `/oidc/logout` | RP-Initiated Logout — `post_logout_redirect_uri` を検証し `state` を反映 |
| POST | `/bypass/token` | 固定 Profile でトークン発行 — `{subject, profile}` → `{access_token, ...}` |
| POST | `/bypass/session` | session を直接作成 — `{subject}` → `{session_id, subject}` |
| GET | `/admin/users` | 固定 User Fixture の一覧 |
| POST | `/admin/reset` | 揮発ストア（code / session）の初期化 |
| POST | `/admin/keys/rotate` | 鍵ローテーション — 契約のみで `501` |

`/bypass/*` ・ `/admin/*` は**dev 専用**: `MOCK_AUTH_DEV_ENDPOINTS=disabled` で `404` を返す。さらに **`NODE_ENV=production` では server が即時終了する**（本番で動かしてはならない）。登録クライアント（`fixtures/clients.json`）は public + PKCE 必須で、`redirect_uri` / `post_logout_redirect_uri` は完全一致で照合する。

## トークン Profile（`/bypass/token`）

再現性のため、異常系は **固定 Profile**（任意 Claim 注入 API ではない）として提供する。Go API は `valid` を受理し、すべての異常系を `401` で拒否することを期待する。

| Profile | 内容 | Go API |
| --- | --- | --- |
| `valid` | 標準 access token（`typ=at+jwt`、正しい `iss`/`aud`/`exp`/`nbf`/`sub`） | 受理 |
| `expired` | `exp` が過去 | 401 |
| `not-yet-valid` | `nbf` が未来 | 401 |
| `wrong-issuer` | `iss` 不一致 | 401 |
| `wrong-audience` | `aud` 不一致 | 401 |
| `missing-subject` | `sub` なし | 401 |
| `invalid-signature` | JWKS に無い鍵で署名（`kid` は同一） | 401 |
| `unsupported-algorithm` | `HS256`（対称鍵・RS256 allowlist 外） | 401 |
| `id-token` | `token_use=id`、`aud=<client_id>`、`typ` が `at+jwt` でない | 401 |

`valid` の access token の Claim はデファクト標準 Profile に従う: `iss` / `sub` / `aud` / `exp` / `iat` / `nbf` / `jti` / `client_id` / `token_use=access` / `scope`。access token は `typ=at+jwt` ヘッダ（RFC 9068）を持ち、Go 検証側はこれで ID Token の誤用を拒否する。

## 鍵 & Fixture

- `keys/mock-key-1.pem` — **固定 RSA 秘密鍵**（再起動しても不変。発行トークンが再現可能）。`kid=mock-key-1`。
- `fixtures/users.json` — サンプルユーザー（Subject / Email / 名前。`status` は未使用）。ファイルが無くても mock は動作する（`/bypass/token` は任意の `subject` を受理）。`fixtures/README.md` を参照。
- `fixtures/clients.json` — 登録済み OAuth クライアント（public client・PKCE 必須・許可する `redirect_uris` / `post_logout_redirect_uris`）。

## 使い方

```sh
docker compose --profile development up mock_auth_server
curl -s http://localhost:4000/.well-known/openid-configuration
curl -s -X POST http://localhost:4000/bypass/token \
  -H 'Content-Type: application/json' \
  -d '{"subject":"user-john-doe","profile":"valid"}'
```

Go API は `AUTH_ISSUER` をホスト URL（`http://localhost:4000`）に、`AUTH_JWKS_URL` を内部サービス URL（`http://mock_auth_server:4000/.well-known/jwks.json`）に向ける — **issuer と JWKS 取得 URL を分離**し、`iss` はブラウザ/ホストから解決可能なまま、鍵取得はコンテナ名を使う。`env/README.md`（Auth セクション）を参照。

## 環境変数

| 変数 | 既定 | 説明 |
| --- | --- | --- |
| `OIDC_PORT` | `4000` | Listen ポート |
| `OIDC_ISSUER` | `http://localhost:4000` | `iss` Claim / OIDC issuer（ホスト解決可能な URL） |
| `OIDC_AUDIENCE` | `go-boilerplate-api` | access token の `aud` Claim |
| `OIDC_CLIENT_ID` | `go-boilerplate-client` | `client_id` Claim / ID Token の audience |
| `MOCK_AUTH_DEV_ENDPOINTS` | `enabled` | `disabled` で `/bypass/*` ・ `/admin/*` を `404` にする |
| `NODE_ENV` | (未設定) | `production` の場合 server を即時終了させる（本番で動かさない） |

## OpenAPI & コード生成

HTTP 表面は `openapi/` で OpenAPI-first に定義し（`openapi/openapi.gen.yaml` にバンドル）、`src/generated/schemas.ts` に [orval](https://orval.dev/) 生成の zod スキーマを持つ。`make gen-mock-auth-oapi` で再生成し、両生成物は commit・CI で drift 検知する。

## 未実装

複数 `kid` の鍵ローテーション（`/admin/keys/rotate` は `501` スタブ）、および Role / `user_identities` / deleted・unknown user の扱い。
