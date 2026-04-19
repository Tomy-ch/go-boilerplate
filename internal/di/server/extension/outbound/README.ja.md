# outbound

[English](README.md) | 日本語

`outbound` は、HTTP レスポンス（出力）側の処理を拡張するための **ミドルウェア DI モジュール群**をまとめたディレクトリです。

リクエスト処理後の **レスポンス変換 / エラーハンドリング / 出力形式の強制 / パニック回復**を担当します。

## モジュール一覧

|モジュール|種別|説明|
|---|---|---|
|`RecoveryModule()`|Use|panic をキャッチし、ログ出力して 500 を返す|
|`ErrorHandlerModule()`|Configurator|アプリケーションエラーを HTTP レスポンスへ統一マッピング|
|`ForceJSONModule()`|Use|レスポンスの Content-Type を JSON に強制|

## 注意点

- Priority は `extension.UseMiddleware` のルールに従い、他のミドルウェアと順序が衝突しないよう調整済み
- Recovery は **最初に動くべきミドルウェアのひとつ**
- ErrorHandler は Echo の `HTTPErrorHandler` を置き換えるため ServeCfg として提供
- outbound ミドルウェアは controller 層であり、**domain / usecase に依存させないこと**
- レスポンス処理の追加は、このディレクトリへ新しい outbound ミドルウェアを追加することを推奨
