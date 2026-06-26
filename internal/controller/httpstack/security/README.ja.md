# security

[English](README.md) | 日本語

セキュリティヘッダミドルウェア（HSTS, X-Frame-Options, Content-Type-Options, Referrer-Policy）です。

## 役割

ブラウザ向けの堅牢化ヘッダは各ハンドラが覚えておくものではなく、すべてのレスポンスに適用すべきベースラインです。これを単一のミドルウェアで設定することで、API 全体に一貫したセキュリティ姿勢を保証し、ポリシーを一箇所で定義・監査可能に保ちます。
