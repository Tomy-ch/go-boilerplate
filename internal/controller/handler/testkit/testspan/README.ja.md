# testspan

[English](README.md) | 日本語

Echo リクエストコンテキストにテスト用トレーススパンを注入します。

## 公開 API

|関数|説明|
|---|---|
|`StartTestSpanForEcho(t, c)`|テストスパンを echo.Context に埋め込み、クリーンアップ関数を返却|
