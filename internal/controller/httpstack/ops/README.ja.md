# ops

[English](README.md) | 日本語

運用系 / インフラエンドポイントを判定します。

## 公開 API

|関数|説明|
|---|---|
|`IsOpsPath(path)`|パスが `/metrics`, `/health`, `/healthz`, `/ready`, `/version` の場合に true を返す|

ログやレートリミットのミドルウェアで ops エンドポイントをスキップするために使用されます。
