# forcejson

[English](README.md) | 日本語

レスポンスの Content-Type を `application/json` に強制します。

## 役割

本 API の契約は JSON のみを扱いますが、個々のハンドラやエラー経路はレスポンスの Content-Type を未設定のまま、あるいは不揃いに残すことがあります。これを単一のミドルウェアで強制することで、すべてのレスポンスが `application/json` を表明することを保証し、各ハンドラが個別に設定しなくてもクライアントは一貫した Content-Type を受け取れます。
