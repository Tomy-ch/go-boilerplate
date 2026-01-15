# cookie ミドルウェア

概要: `internal/controller/httpstack/cookie` パッケージは、HTTP レスポンスの `Set-Cookie` ヘッダを書き換え・正規化するための機能を提供します。Echo 用ミドルウェアでレスポンス書き換えラッパーを挿入し、セキュアな Cookie ポリシー（Secure/HttpOnly/SameSite/Path/Domain/Max-Age 等）の強制や不要な属性の調整を行います。

## 役割

- Echo ミドルウェア `Middleware(cfg *SecurityCookie)` を提供し、レスポンスの `Set-Cookie` を `SecurityCookie` 設定に従って上書きします。
- `SecurityCookie` は設定値（`Secure`/`HttpOnly`/`SameSite`/`Path`/`Domain`/`Max-Age` など）を取り込み、個別クッキー名の選択・除外も可能にします。
- `cookie` ヘッダのパース/構築ユーティリティ（`parseSetCookie` / `buildSetCookie`）を含み、属性の順序をある程度保持して出力します。
- レスポンス書き換えは `cookieRewriteWriter` によって行われ、`WriteHeader` / `Hijack` / `Flush` / `ReadFrom` 等のインターフェースを透過します。

## 必要度

### 本番運用での必須度

- 必須度: 本番運用で推奨

- 理由: ブラウザのセキュリティ要件（SameSite/Secure/HttpOnly 等）やドメイン/Path ポリシーを一貫して適用することで、セッションの安全性と互換性を高められるため、本番では推奨されます。

### 開発/検証での必須度

- 必須度: 開発/検証で推奨

- 理由: Cookie の属性に起因する挙動差（クロスサイト送信やブラウザ依存のハンドリング）を確認する際に有用です。開発環境では柔軟に設定を変えて検証してください。

### 無効化した場合の影響

- ミドルウェアを無効化すると、アプリケーションが Set-Cookie を任意の形式で返すことになり、ブラウザ側で期待するセキュリティ動作が保証されなくなります（SameSite=None の場合の Secure 必須など）。

## 注意点

- `SecurityCookie` の設定は環境やプロダクト要件に合わせて慎重に調整してください。特に `SameSite=None` と `Secure` の組合せはブラウザ仕様に基づいた扱いが必要です。
- ミドルウェアはレスポンスの `ResponseWriter` をラップして `Set-Cookie` を書き換えます。WebSocket 等で `Hijack` が用いられる場合でも内部で対応していますが、ミドルウェア順序や他ミドルウェアとの相互作用に注意してください。
- `cookie.RealIP()` 相当の依存はないが、`__Host-` / `__Secure-` プレフィックスの扱い（強制 Secure / Path=/ / Domain 削除）は `RewriteSetCookie` 内で行われます。

## 実装の要点

- ミドルウェア: `Middleware(cfg *SecurityCookie)` が `cookieRewriteWriter` を挿入してレスポンスを書き換えます。
- レスポンスラッパー: `cookieRewriteWriter` はヘッダを内部で保持し、`WriteHeader` 時に `Set-Cookie` を `SecurityCookie.RewriteSetCookie` で上書きしてから下流に渡します。`Hijack` / `Push` / `ReadFrom` 等の最適化経路も透過します。
- Cookie パーサ/ビルダー: `parseSetCookie` は Set-Cookie をパースして属性を map と order スライスで保持し、`buildSetCookie` が再構築します。解析失敗時は空文字を返し、呼び出し側は元の値を保持する挙動です。
- セキュリティ設定: `SecurityCookie` はコンストラクタ `NewSecurityCookie(p *config.SecureCookieConfig)` により構築され、デフォルトで `HttpOnly` を付与、`forcePath` に `/` をセットする等の安全側のデフォルトを持ちます。`RewriteSetCookie` は名前ベースの除外/適用や `__Host-`/`__Secure-` プレフィックス強制にも対応します。

## 使い方

ミドルウェア登録は、`internal/di/module/`で行います。

## 前提 / 要件

- `config.SecureCookieConfig` から必要な設定（SameSite, Secure, Domain など）を取得できること。
- ミドルウェアを利用する際は、他のミドルウェア（認証/セッション等）との順序に注意してください。

## トラブルシューティング

- クッキー属性が想定どおりに反映されない: `SecurityCookie` の設定（SameSite, Secure, Path, Domain, Max-Age）を確認してください。
- WebSocket ハンドシェイクで Set-Cookie が反映されない: `cookieRewriteWriter` は `Hijack` をサポートしていますが、ハンドシェイク実装が `ResponseWriter` のヘッダをどの時点で参照するかに依存します。必要なら `Hijack` パスの動作を確認してください。
- SameSite=None のときクッキーが送信されない: ブラウザは SameSite=None に Secure が必須です。`SecurityCookie` の `forceSameSite` と `forceSecure` の設定を確認してください。

## モック / テスト

- `mock/` ディレクトリに自動生成のモックがある場合は、ミドルウェアや依存を差し替えて単体テストを行えます（ここでは詳細なテスト記載は省略します）。

追加でサンプル設定や `config.SecureCookieConfig` の例を README に入れることもできます。必要なら追加します。
