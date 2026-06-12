# testecho

[English](README.md) | 日本語

Echo ハンドラテスト用のビルダーパターン HTTP テストクライアントです。

## 公開 API

|メソッド|説明|
|---|---|
|`NewEchoTestClient(t, e)`|テストクライアントを作成|
|`WithAppErrorHandler()`|本番相当のエラーハンドラを設定（Echo の `HTTPErrorHandler` を上書き）|
|`Method(m)`|HTTP メソッドを設定|
|`RoutePattern(p)`|ルートパターンを設定（例: `/users/:id`）|
|`RequestURL(u)`|実際のリクエスト URL を設定|
|`JSONBody(v)`|JSON リクエストボディを設定|
|`RawBody(r, contentType)`|生のリクエストボディを設定|
|`Header(k, v)`|リクエストヘッダーを設定|
|`AuthBearer(token)`|Bearer トークンを設定|
|`PathParams(params)`|パスパラメータを設定|
|`QueryParams(params)`|クエリパラメータを設定|
|`Build()`|Request / ResponseRecorder / echo.Context を返却|
|`Serve()`|Echo にリクエストを送信し ResponseRecorder を返却|
