# idempotency

[English](README.md) | 日本語

`Idempotency-Key` ベースのリクエスト重複排除の入り口です。oapi-codegen の StrictMiddleware スロット用に提供され、`e.Use` では登録しません。

## 役割

安全なリトライを実現するには、クライアントが同一の更新系リクエストを再送したことをサーバが認識し、重複によって操作が二重に適用されないようにする必要があります。この認識をミドルウェア境界（リクエストが既に型付き表現へパースされている地点）に置くことで、各ハンドラがキー処理やリクエスト指紋計算を再実装することなく、すべてのエンドポイントが一様に冪等性を選択できます。本パッケージはリクエストに対して冪等性コンテキストを確立するだけであり、保存済みレスポンスの永続化・再生といった実処理は usecase 層が担います。

## 補足

- `Middleware()` は StrictMiddleware の構造的シグネチャ `func(next NextFunc, operationID string) NextFunc` で入り口を返します（`NextFunc` は `func(ctx echo.Context, request any) (any, error)`）。`StrictMiddleware[H]()` はそれをパッケージ固有の oapi-codegen `StrictMiddlewareFunc` 型（例: `gen.StrictHandlerFunc`）へ適合させるため、生成された strict handler のミドルウェアスロットに登録されます。`e.Use` 経由では登録しません。
- `Idempotency-Key` ヘッダが無い場合、リクエストはそのまま素通しされます（非冪等として扱います）。
- 冪等性は認証済みリクエストに対してのみ発動します。スコープキーに認証済みの `Subject` を用いるため、リクエストコンテキストに認証プリンシパルが存在しない場合は素通しされます。
- キー検証は違反キーを `400`（`apperror.ErrInvalidArgument`）で拒否します。キーは非空・255 バイト以下・印字可能 ASCII のみで構成されている必要があります。
- リクエスト指紋は `method + path + JSON(型付き request)` の SHA-256 です。request の marshal は fail-closed とし、失敗時は弱い指紋で処理を続行せず内部エラー（`apperror.ErrInternal`）を返します。
- 成功時、ミドルウェアは usecase の `WithRequest` を介して `idempotency.Request`（scope・key・fingerprint・method・path・operationID）をリクエストコンテキストへ格納し、次のハンドラへ委譲します。以降の処理は usecase 層が消費します。
