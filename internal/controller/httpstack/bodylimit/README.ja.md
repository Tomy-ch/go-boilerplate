# bodylimit

[English](README.md) | 日本語

リクエストボディのサイズを上限化します（`SERVER_BODY_LIMIT_MB`、MB 単位）。

## 役割

過大なリクエストボディによるメモリ圧迫・DoS 面を抑えます。`Middleware(limitMB int)` は echo 標準の `middleware.BodyLimit` を薄くラップし、MB 値を echo のバイト文字式（`"%dM"`。gommon/echo は `1M` を 1,000,000 バイト＝10進として扱う）へ変換します。上限超過時は echo が **413 Request Entity Too Large** を返します。

## 補足

- **Pre** ミドルウェア（priority 2）として登録します（`internal/di/server/extension/inbound` 参照）。echo の `BodyLimit` はボディ reader をラップするだけでルーティング非依存なため、`Pre` に置けば OpenAPI validator（`requestBody` を decode/buffer する `Use` ミドルウェア）がボディを読む**前**に確実に上限が効きます。後ろに置くと validated な `POST`/`PUT` で validator が無制限ボディを読み切り、上限がサイレントに無効化されます。
- 上限は **MB の整数**（`ServerConfig.BodyLimitMB()`）で設定し、自由形式の文字列にしないことで単位を明確にし、不正値は config 読込時に弾きます。
- outbound 側（レスポンスボディ）は HTTP クライアントの `MaxResponseBytes` で既に上限化済みで、本パッケージは inbound の穴を塞ぎます。
