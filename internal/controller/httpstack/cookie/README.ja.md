# cookie

[English](README.md) | 日本語

Set-Cookie ヘッダのセキュリティ属性（Secure / HttpOnly / SameSite / Path / Domain / Max-Age）を強制するミドルウェアです。

## 役割

Cookie のセキュリティ属性は各ハンドラが付け忘れたり不揃いに設定したりしがちです。本パッケージは送出される `Set-Cookie` ヘッダを単一のミドルウェアで書き換えることで、すべてのレスポンスに一貫した Cookie セキュリティポリシーを保証し、ハンドラ側は堅牢化フラグを毎回書き直すことなく Cookie を設定でき、ポリシーを一箇所に集約できます。
