# cookie

[English](README.md) | 日本語

Set-Cookie ヘッダのセキュリティ属性（Secure / HttpOnly / SameSite / Path / Domain / Max-Age）を強制するミドルウェアです。

## 役割

Cookie のセキュリティ属性は各ハンドラが付け忘れたり不揃いに設定したりしがちです。本パッケージは送出される `Set-Cookie` ヘッダを単一のミドルウェアで書き換えることで、すべてのレスポンスに一貫した Cookie セキュリティポリシーを保証し、ハンドラ側は堅牢化フラグを毎回書き直すことなく Cookie を設定でき、ポリシーを一箇所に集約できます。

## `SECURE_COOKIE_SAME_SITE` の clamp（安全な既定値であり、silent ではない）

`normalizeSameSite` は `Lax` / `Strict` / `None`（大文字小文字を区別しない）のみを受け付けます。**それ以外の値 — 空文字列を含む — は「上書きしない」へ clamp** され、起動失敗にはせず、フレームワーク/既定の `SameSite` をそのまま残します。これは意図的な回復性の選択であり、`SECURE_COOKIE_SAME_SITE` のタイプミスは値レベルで上書きを silent に弱めます。挙動をレビュー可能に保つため、ここに記し、セットアップレビュー（[`docs/get-started/setup-repository.md`](../../../../docs/get-started/setup-repository.md)）にも列挙します。特定のポリシーを強制したい場合は、この変数を上記 3 つの受理値のいずれかに設定してください。
