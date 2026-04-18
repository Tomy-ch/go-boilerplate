# testassert

[English](README.md) | 日本語

Controller 層テスト用のアサーションヘルパーです。

## 公開 API

|関数|説明|
|---|---|
|`AssertJSONEqual[T](t, expectedCode, expectedResponse, actualResponse)`|HTTP ステータスコードと JSON ボディを検証|
|`AssertEchoRouterMethods(t, expectedMethods, actualRoute)`|登録されたルートの HTTP メソッドを検証|
|`AssertEchoRouterPath(t, expectedPath, actualRoute)`|登録されたルートのパスを検証|
