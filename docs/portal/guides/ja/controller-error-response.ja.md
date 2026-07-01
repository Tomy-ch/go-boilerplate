# error response

[English](README.md) | 日本語

このパッケージは、HTTPエラーレスポンスの生成と管理を行います。

## 役割

- **統一フォーマットの提供**: APIで返すエラーレスポンスを一貫した構造で返し、クライアント側の取り扱いを簡単にします
- **ハンドラ / ミドルウェアからの再利用**: 各エンドポイントのハンドラやミドルウェアは、公開関数を呼び出してエラーを返します
- **型安全なレスポンス**: `gen/type.gen.go`（OpenAPI から自動生成）の型を利用してレスポンスを組み立てます

## ファイル構成

|ファイル|役割|
|---|---|
|`error_response.go`|レスポンス組み立て、公開関数|
|`http_error.go`|エラーコード / メッセージ定数、apperror → HTTP マッピング|
|`gen/type.gen.go`|OpenAPI から自動生成された型定義（編集不可）|

## エラーレスポンスの構造

```json
{
  "code": "INTERNAL_ERROR",
  "message": "サーバーで予期しないエラーが発生しました。時間をおいて再度お試しください。",
  "details": ["具体的なエラーの説明"],
  "requestId": "123e4567-e89b-12d3-a456-426614174000"
}
```

|フィールド|説明|
|---|---|
|`code`|アプリケーションレベルのエラー識別子|
|`message`|エンドユーザー向けのメッセージ|
|`details`|デバッグ用の追加情報（公開可能なもののみ）|
|`requestId`|リクエスト追跡用のユニークID|

HTTPステータスコードはレスポンスヘッダで返し、スタックトレース等の内部情報はログにのみ出力します。

## 公開関数

|関数|説明|
|---|---|
|`NewHTTPErrorFromAppError`|`apperror` から対応するHTTPエラーレスポンスを生成（未知のエラーは 500 にフォールバック）|
|`NewHTTPErrorFromStatus`|HTTPステータスコードから対応するエラーレスポンスを生成（未知のステータスは 500 にフォールバック）|

## エラーコードとHTTPステータスの対応

### apperror → HTTP マッピング

|apperror|HTTPステータス|エラーコード|
|---|---|---|
|`ErrInvalidArgument`|400 Bad Request|`BAD_REQUEST`|
|`ErrUnauthenticated`|401 Unauthorized|`UNAUTHORIZED`|
|`ErrPermissionDenied`|403 Forbidden|`ACCESS_DENIED`|
|`ErrNotFound`|404 Not Found|`NOT_FOUND`|
|`ErrConflict`|409 Conflict|`RESOURCE_CONFLICT`|
|`ErrValidation`|422 Unprocessable Entity|`VALIDATION_FAILED`|
|`ErrTooManyRequests`|429 Too Many Requests|`TOO_MANY_REQUESTS`|
|`ErrCanceled`|499 Client Closed Request|`CLIENT_CLOSED_REQUEST`|
|`ErrUnimplemented`|501 Not Implemented|`NOT_IMPLEMENTED`|
|`ErrUnavailable`|503 Service Unavailable|`SERVICE_UNAVAILABLE`|
|その他|500 Internal Server Error|`INTERNAL_ERROR`|

### エラーコード一覧

|コード|デフォルトメッセージ|
|---|---|
|`BAD_REQUEST`|入力内容に誤りがあります。再度ご確認ください。|
|`UNAUTHORIZED`|ログインが必要です。ログインして再度お試しください。|
|`ACCESS_DENIED`|この操作を行う権限がありません。|
|`NOT_FOUND`|お探しの情報が見つかりませんでした。|
|`RESOURCE_CONFLICT`|既に同じ情報が登録されています。|
|`VALIDATION_FAILED`|入力内容の検証に失敗しました。修正して再度お試しください。|
|`TOO_MANY_REQUESTS`|リクエストが多すぎます。しばらくしてから再度お試しください。|
|`CLIENT_CLOSED_REQUEST`|リクエストがキャンセルされました。|
|`INTERNAL_ERROR`|サーバーで予期しないエラーが発生しました。時間をおいて再度お試しください。|
|`NOT_IMPLEMENTED`|この機能は提供されていません。|
|`SERVICE_UNAVAILABLE`|現在この機能はご利用いただけません。しばらくしてから再度お試しください。|

## 設定変更方法

- **レスポンスロジックの修正**: `error_response.go` を編集
- **エラーコード / メッセージの追加**: `http_error.go` に定数とマッピングを追加
- **型定義の更新**: OpenAPI 仕様を更新後、`make gen-api` で再生成

## 注意点

- `Details` にスタックトレースや機密情報を含めないこと。内部情報はログにのみ出力する
- エラーコードとHTTPステータスの整合性を保つこと
- `gen/` 配下のファイルは手動で編集しないこと
