# jwt（JWT 認証）

[English](README.md) | 日本語

> このファイルは canonical な [README.md](README.md) の日本語訳です。内容の更新は canonical 側で行い、本ファイルへ同期してください。

このディレクトリは、アクセストークン（JWT）を検証する `Authenticator` 実装を提供します。署名鍵は**固定 RSA 公開鍵**（`New`）または **JWKS エンドポイントからの `kid` 動的解決**（`NewJWKS`）のいずれかで解決します。開発専用の `local` 実装に対する本番向けの対になる実装であり、**デファクト標準の検証コア**のみを扱います。

## 役割

- JWT として提示されたアクセストークンの署名と標準クレームを検証する
- 検証成功時に検証済みの `Authn`（subject + scopes + 素のクレーム）を生成する
- すべての検証失敗を `apperror.ErrUnauthenticated` へ正規化する（fail-closed）

## 検証スコープ（標準コア）

本実装は意図的に**デファクト標準プロファイル**のみをサポートします。

- **アルゴリズム allowlist**（既定 `["RS256"]`、注入可能）による非対称署名検証。鍵混同攻撃を防ぐため `alg=none` および `HS256` 等の対称アルゴリズムは常に拒否します。
- 固定 RSA 公開鍵による署名検証（PEM はコンストラクタでパースし、パース失敗は設定エラー）。
- クレーム検証: `iss` / `aud` / `exp` / `nbf` / `sub`。`exp` は必須、`aud` も必須（標準プロファイル）。
- 注入可能な `Leeway`（既定 60s）によるクロックずれ許容。
- 任意の `typ` ヘッダ検証（`ExpectedType`、例: RFC 9068 の `at+jwt`）で ID Token の誤用を拒否。空なら無効。
- OAuth2 標準 `scope` クレーム（スペース区切り文字列 → `[]string`）からのスコープ抽出。

`exp` / `nbf` 検証をテストで決定的にするため、時刻は `clock.Clock` バウンダリ経由で注入します。

## コンストラクタ

署名鍵は注入された `keyResolver` 経由で解決し、クレーム検証ロジックは全コンストラクタで共通です。

- `New(Params)` — 固定 RSA 公開鍵。`Params` は検証パラメータ（公開鍵 PEM・issuer・audience・許可アルゴリズム・leeway・期待 typ・clock）を保持します。PEM が不正な場合に失敗します。
- `NewJWKS(JWKSParams, httpclient.Client)` — JWKS エンドポイントからの動的鍵解決（`kid` 参照・TTL キャッシュ・遅延取得）。JWK Set のパースは [`github.com/go-jose/go-jose/v4`](https://github.com/go-jose/go-jose)、取得は resilient な `httpclient` substrate 経由（infrastructure 層は `net/http` 直 import 禁止）。HTTP タイムアウト / リトライ / circuit breaker / budget は `jwks` downstream プロファイル（`NewDownstreamProfile`）由来で、param には持ちません。`JWKSParams` は `Params` を埋め込み（PublicKeyPEM は不使用）、JWKS URL とキャッシュ TTL を追加します。取得は遅延（初回利用・cache miss 時）のため、バックグラウンド goroutine や lifecycle 束ねは不要です。
- `NewWithKeyfunc(Params, keyResolver)` — 任意の `jwt.Keyfunc` を直接受ける下層の口（JWKS backed の解決やテストダブルの注入に使用）。

必須パラメータ（clock / issuer / audience）が欠けている場合は生成に失敗します。これらは認証エラーとは区別される設定エラーです。

## エラーハンドリング

すべての検証失敗は `ErrJWTAuthenticatorInvalidToken` センチネルへ正規化されます。このセンチネルは `apperror.ErrUnauthenticated` をラップします（fail-closed）。原因（署名不一致 / `exp` / `iss` / `aud` / `typ` 等）はエラーチェーンに保持されるため、運用側はログ・トレースから失敗要因を切り分けられます。一方で呼び出し側が受け取るのは常に正規化された `401` のみです。これは `internal/infrastructure/rdb/pgerror` の `pgerror.NormalizeError` 規約に倣っています。

## 拡張ポイント

以下は標準コアの**対象外**であり、利用側の IdP が必要とする場合にテンプレート利用者が追加します。

- Cognito のアクセストークン方言（`token_use=access` 検証、`aud`→`client_id` 読み替え — Cognito のアクセストークンは `aud` を持たない）
- Azure AD の `scp` / `roles` クレーム
- 楕円曲線鍵（`ES256` 等） — 現行コンストラクタは RSA 公開鍵をパースする
- opaque（非 JWT）アクセストークン — 対象外。これらの IdP は本実装ではサポートしない

## 補足

- JWKS 鍵解決（`NewJWKS`）は固定鍵経路と同じ標準クレーム集合を検証し、異なるのは鍵ソースのみです。JWK パースは `go-jose/v4`、取得は `httpclient` substrate に委譲し、`kid` 参照と TTL キャッシュはパッケージ内に保持、その上で RSA 署名方式ガード（鍵混同防御）を適用します。複数 `kid` のローテーションは後続フェーズです。
- 内部ユーザー ID の解決（`sub` → DB 経由のアプリケーションユーザー ID）は別関心事で、identity 解決 Phase が担います。本実装は検証済みだが未解決の `Authn` を返します。
