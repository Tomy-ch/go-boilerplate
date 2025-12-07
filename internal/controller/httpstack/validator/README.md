# `validator` パッケージ

概要: OpenAPI 仕様に基づくリクエスト検証ミドルウェアを提供します。`oapi-codegen` によって埋め込まれた OpenAPI スキーマを読み込み、`oapi-codegen/echo-middleware` のリクエストバリデータを利用します。

## 役割

`validator` は以下を提供します。

- `Middleware(validator *openapi3.T) echo.MiddlewareFunc` : OpenAPI 仕様に基づいたリクエスト検証ミドルウェアを返します（内部で `middleware.OapiRequestValidator` を使用）。
- `GetValidator() (*openapi3.T, error)` : 埋め込みの OpenAPI スペックをデコードして `openapi3.T` を返します（`gen.GetSwagger()` を利用）。

主要ファイル:

- `validator.go` : ミドルウェアとバリデータ取得の実装。
- `gen/validate.gen.go` : `oapi-codegen` によって生成された埋め込み仕様ファイル（Swagger/OpenAPI スペックを含む）。
- `validator_test.go` : 動作を検証する単体テスト。

## 必要度

### 本番運用での必須度

- 必須度: 本番運用で推奨（API を公開する場合はほぼ必須）

理由: リクエスト検証により不正な入力を早期に弾き、内部処理の堅牢性とセキュリティを向上できます。外部に公開する API では仕様に沿った検証は重要です。

### 開発/テスト運用での必須度

- 必須度: 開発/テスト運用で推奨

理由: 開発段階で仕様違反を早期に検出できるため、テストの信頼性が向上します。OpenAPI を仕様書としても使っている場合は特に有用です。

### 無効化した場合の影響

ミドルウェアを適用しない場合、リクエストの型や必須フィールドのチェックが行われず、内部で予期せぬエラーが発生するリスクが高まります。クライアント側の誤ったリクエストをそのまま進めてしまう可能性があります。

注意: `gen/validate.gen.go` は `//go:generate` コメントに基づいて生成されます。OpenAPI 仕様を更新した場合は `oapi-codegen` を再実行して生成ファイルを更新してください。

## 注意点

- `GetValidator()` は埋め込みされたスペックを返すため、実行バイナリに仕様が含まれます。仕様を更新したら生成ファイルを確実に更新してください。
- バリデーションはリクエストレベルの検証であり、ビジネスロジック上の追加検証は別途行ってください。
- `oapi-codegen` / `kin-openapi` のバージョンや振る舞いによって検証の挙動が変わることがあるため、依存関係の管理に注意してください。

## 作成・更新

作成・更新: `internal/controller/httpstack/validator` の実装に合わせて記述。
