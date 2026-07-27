# errorhandler

[English](README.md) | 日本語

Echo / OpenAPI バリデーション / アプリケーションレベルのエラーを統一的な JSON レスポンスに正規化し、構造化ログとともに処理する統一 HTTP エラーハンドラです。

## アーキテクチャ

```mermaid
flowchart TB
    Error["エラー発生"]
    Guard{"処理済み?"}
    Normalize["normalizeHTTPError"]
    TypeCheck{"エラー型?"}
    AppErr["HTTPErrorResponse (apperror)"]
    EchoErr["ステータスを持つエラー<br/>(echo.HTTPError / 定義済み / OpenAPI 検証)"]
    EchoNorm["normalizeEchoHTTPError"]
    Fallback["NewHTTPErrorFromAppError (フォールバック)"]
    AddReqID["RequestID 付与"]
    Gate{"details ゲート<br/>(policy.Allows?)"}
    Write["JSON レスポンス書き込み<br/>(未 opt-in なら details を削除)"]
    Log["構造化ログ出力<br/>(details は温存)"]

    Error --> Guard
    Guard -- yes --> return
    Guard -- no --> Normalize
    Normalize --> TypeCheck
    TypeCheck -- HTTPErrorResponse --> AppErr --> AddReqID
    TypeCheck -- ステータスを持つエラー --> EchoErr --> EchoNorm --> AddReqID
    TypeCheck -- その他 --> Fallback --> AddReqID
    AddReqID --> Gate --> Write --> Log
```

## エラー正規化

ハンドラは以下の優先順位でエラーを処理します。

### 1. `response.HTTPErrorResponse`（アプリケーションエラー）

ハンドラ内で `response.NewHTTPErrorFromAppError()` によりラップ済みのエラー。

- HTTP ステータスが有効（400-599）: そのまま使用し、RequestID を付与
- HTTP ステータスが無効: `NewHTTPErrorFromAppError(internal)` で再正規化

### 2. HTTP ステータスを持つエラー（Echo / OpenAPI エラー）

ステータスは `echo.StatusCode` で解決するため、`echo.HTTPError` に加えて Echo の定義済み
エラー（`echo.ErrNotFound` など。型は非公開）も対象になります。OpenAPI バリデーションの
失敗もこの経路に入ります —— バリデーションミドルウェアが決めたステータス（不正なリクエスト
は 400、経路解決不能は 404 / 405、資格情報の拒否は 401。`oapi/auth` の README 参照）を持つ
`echo.HTTPError` として届きます。

400〜599 の範囲外のステータスはエラーステータスとみなさず、フォールバックへ落ちます。

### 3. フォールバック

認識できないエラーは `response.NewHTTPErrorFromAppError()` に渡され、`apperror` の型に基づいて HTTP ステータスコードにマッピングされます。

## レスポンス形式

すべてのエラーは `response.HTTPErrorResponse` を使って JSON で返されます：

```json
{
  "code": "BAD_REQUEST",
  "message": "...",
  "details": ["..."],
  "requestId": "..."
}
```

- `requestId` は常に付与（`requestid.GetRequestIDFromResponse` で取得）
- `Details` と `Internal` エラーは利用可能な場合に含まれる
- エラーが `apperror.Meta` を運んでいる場合、`NewHTTPErrorFromAppError` 内で `code` / `message` / `details` がステータス既定値を上書きする（HTTP ステータスは変わらない）— [`controller/error/response/README.ja.md`](../../error/response/README.ja.md) の「`apperror.Meta` による上書き」節を参照
- `Internal` エラーとスタックトレースはログに出力されるが、**クライアントには返されない**

### details の opt-in ゲート（fail-closed）

`details` は**エンドポイントごとの opt-in**。`DetailPolicy`（起動時に OpenAPI spec から構築、
`detail_exposure.go`）が「どの operation が `ErrorResponseWithDetails` スキーマを宣言しているか」を
前計算する。エラー経路で、レスポンスが `details` を持つ場合、`handleHTTPError` はリクエストの
operation を解決し、opt-in していない限り**クライアント wire からのみ** `details` を落とす
（`writeErrorResponse` が body をコピー。`resp` 本体とログには完全な `details` が残る）。ルート
不一致・未 opt-in はいずれも **fail-closed**（details なし）。policy 用 router は servers を除去した
spec 複製から作るため Host 非依存で、proxy / test の Host でもパス + メソッドで解決できる。
理由: [ADR-0041](../../../../docs/adr/0041-error-details-opt-in-gate.md)。

## ログ出力

エラーログは `ObservabilityConfig.TargetStatusCodeSet()` で制御されます：

- 設定されたステータスコードのみログ対象
- **5xx**: `Error` レベル（`errorhandler.server_error`）
- **4xx**: `Warn` レベル（`errorhandler.client_error`）

ログフィールド：

- HTTP ステータス、エラーコード、エラーメッセージ、RequestID
- リクエスト詳細（メソッド、パス、URI、リモート IP、ホスト、ユーザーエージェント等）
- クエリパラメータ、パスパラメータ
- Trace ID / Span ID（Observability 有効時）
- 内部エラーメッセージとスタックトレース（デバッグ用）

## 再入ガード

初回呼び出し時にハンドラは `ctxhelper.SetErrorHandledToEcho(c, true)` を呼び、以降の呼び出しは `ctxhelper.GetErrorHandledFromEcho(c)` の判定で早期 return します。これによりエラーレスポンス書き込み中に再度エラーが起きても無限再帰しません。フラグは Echo の内部ストアではなく `scripts/genctxkey` が生成する typed sentinel として request の context 側に保持されます。

## リカバリミドルウェアとの連携

上流の `recovery` ミドルウェアが既にパニックをログ済みの場合、同じコンテキストには `ctxhelper.SetRecoveredToEcho(c, true)` で `Recovered` sentinel が立っています。本ハンドラは `logHTTPError` を呼ぶ前に `ctxhelper.GetRecoveredFromEcho(c)` をチェックし、ログ二重出力を抑止します（500 レスポンス自体は返します）。

## ファイル構成

|ファイル|責務|
|---|---|
|`http_error_handler.go`|メインハンドラ、正規化ディスパッチ、ログ出力|
|`echo_http_error_handler.go`|HTTP ステータスを持つエラー → `HTTPErrorResponse` の正規化|
|`detail_exposure.go`|`DetailPolicy` — OpenAPI spec から解決するエンドポイントごとの `details` opt-in|

## カバレッジ例外

`docs/testing-conventions.md` §9 に基づき、以下の infallible な防御分岐は未カバーのまま残す(作為的テストは書かない):

- `http_error_handler.go` `handleHTTPError` — `writeErrorResponse` が失敗しつつレスポンス未 commit のときだけ通る入れ子 `WriteHeader(500)`。body は常に JSON 化可能な固定 struct(`gen.ErrorResponseWithDetails`)なので `c.JSON` は書き込み中(= `WriteHeader` で commit 済みの後)にしか失敗できず、未 commit での失敗は到達不能。到達可能な書き込み失敗経路(ログ出力・二重 commit なし)はカバー済み。

## 注意点

- エラーレスポンスの書き込みに失敗した場合、フォールバックとして `500` ステータスを返し、書き込みエラーをログに出力
- エラーレスポンスは `controller/error/response/` の `response.HTTPErrorResponse` を使用 — エラーコードとメッセージのマッピングはそちらを参照
- このハンドラは Echo のデフォルトエラーハンドラを完全に置き換える
