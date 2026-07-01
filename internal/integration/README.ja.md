## integration ディレクトリ

[English](README.md) | 日本語

このディレクトリは **結合テスト (integration test)** をまとめるための場所です。  

ユニットテストでは拾いきれない、Echo サーバを立ち上げて **実際の HTTP 通信経路** を通した検証を行います。

このテストは **DB や Infrastructure を含む統合テストではなく、HTTP 境界の検証を目的とした結合テスト**です。

## テスト戦略

本プロジェクトでは以下のテスト戦略を採用しています。

- `Domain` → Unit Test
- `Usecase` → Unit Test
- `Controller` → Unit Test
- `Integration` → HTTP boundary test

integration テストでは **HTTP 経路全体の動作確認**のみを行います。

つまり次の範囲を検証します。

```mermaid
flowchart TB
    Router --> Middleware --> Handler --> Presenter["Presenter / Response serialization"]
```

以下は **integration テストでは扱いません**

- Domain ロジック
- Usecase 内部ロジック
- Repository 実装
- DB 接続
- SQL 実行

## テストレベルの定義(Test Pyramid)

本プロジェクトでは **Test Pyramid** を前提としたテスト戦略を採用しています。

```mermaid
flowchart TB
    Domain["Domain Unit"]
    Usecase["Usecase Unit"]
    Controller["Controller Unit"]
    Integration["Integration"]
    E2E["E2E (なし)"]

    Domain --> Usecase --> Controller --> Integration --> E2E
```

方針：

- **Domain / Usecase の Unit Test を中心にテストを書く**
- Integration Test は **HTTP 境界の検証のみ**
- E2E テストはこのプロジェクトでは扱いません

この方針により次のメリットを得ます。

- テスト実行速度の高速化
- テストの保守性向上
- 失敗時の原因特定の容易さ

## なぜ Usecase を mock するのか

integration テストでは **Usecase を mock 化**します。

理由は **レイヤ境界を守るため**です。

integration テストの目的は **HTTP Layer の検証**であり、  
以下の範囲のみを対象とします。

```mermaid
flowchart TB
    Router --> Middleware --> Handler --> Response["Response serialization"]
```

Usecase / Domain / Repository のロジックは  
それぞれ **Unit Test の責務**です。

もし Usecase を実装のまま利用すると

- DB
- Repository
- Domain

までテスト範囲が拡大してしまい、  
**HTTP boundary test の目的が崩れてしまいます。**

## Integration Test の配置ポリシー

integration テストは **公開 API の動作確認**として配置します。

対象となるエンドポイント例：

- `/health`
- `/healthz`
- `/ready`
- `/version`
- `/v1/...`

つまり、**公開される HTTP API** のみを integration テストの対象とします。

内部関数や handler の細かなロジックは  
**Controller Unit Test で検証します。**

## integration ディレクトリがある理由

### 層の分離

ユニットテストは個々の関数やハンドラを対象としますが、  
結合テストは **ルータ〜ミドルウェア〜ハンドラ〜レスポンス整形までの一連の流れ** を確認します。

このディレクトリを分けることで、**テストの目的と粒度を明確にしています。**

- `internal/controller/handler/...` → Handler Unit Test
- `internal/integration` → HTTP Integration Test

### 実運用に近い検証

integration テストでは **実際に HTTP リクエストを送信**して検証します。

これにより以下を確認できます。

- Router のバインド
- Middleware の適用
- Request → DTO 変換
- Response serialization
- HTTP ステータスコード

CI/CD やスモークテストとしても利用できます。

## 統合テストの範囲

integration テストで扱う範囲は次の通りです。

```mermaid
flowchart TB
    Router["Echo Router"] --> Middleware --> Handler --> Response
```

Usecase は **mock を利用します。**

理由：

integration テストの目的は **HTTP boundary の検証**であり  
アプリケーションロジックの検証ではないためです。

例

- `mock_user.NewMockUsecase`
- `mock_healthcheck.NewMockUsecase`

## テストの流れ

integration テストは次の手順で実行されます。

```mermaid
flowchart TB
    New["Echo.New()"] --> Bind["BindHandler"] --> Start["StartServer"] --> Req["HTTP Request"] --> Assert["Assert Response"]
```

具体例

```mermaid
flowchart TB
    New["echo.New()"] --> Bind["handler.BindHandler()"] --> Start["StartServer()"] --> Do["DoJSON()"] --> Assert["AssertJSONResponseType()"]
```

## integration_test.go で定義されている関数

### `StartServer(t *testing.T, e *echo.Echo) *Server`

Echo を `httptest.NewServer` で立ち上げ、結合テスト用の簡易サーバを返します。

特徴

- `t.Cleanup` によりサーバは自動停止
- HTTP クライアントを内部で保持
- テストヘルパー関数 `Do` / `DoJSON` を提供

使用例

```go
e := echo.New()
handler.BindHandler(e)

srv := StartServer(t, e)
```

### `StopServer()`

明示的にサーバを停止します。

通常は `StartServer` 内の `t.Cleanup` により自動停止されるため、  
特別なケース以外では使用しません。

### `Do(method, path string, reqBody io.Reader, contentType string, headers http.Header)`

任意の HTTP リクエストを送信します。

機能

- HTTP method 指定
- request body 指定
- header 指定
- content-type 指定

内部では `http.NewRequestWithContext` を使用しています。

### `DoJSON(method, path string, reqBody any, headers http.Header)`

JSON 用のショートカット関数です。

特徴

- `reqBody` を JSON encode
- `Content-Type: application/json` を自動設定
- 内部的に `Do` を呼び出す

例

```go
actual := srv.DoJSON(http.MethodGet, "/health", nil, nil)
```

### `AssertJSONResponseType[T any]`

HTTP 境界の到達確認アサーションです。レスポンスが HTTP 経路を通り、期待した型へ
シリアライズされることを検証します。**個々のフィールド値の正しさは検証しません。**

検証内容

- HTTP Status Code = 200
- Content-Type = application/json
- レスポンスボディが型 `T` に Unmarshal 可能

このヘルパーは意図的に値比較を行いません。上記のテストピラミッドのとおり、レスポンス値の
正しさ（Presenter のフィールドマッピング）は **Controller Unit Test** が独立したオラクルで
検証する責務です。ここで値比較を重複させると integration テストが Presenter の詳細に結合し
壊れやすくなります。build info や `RegisteredAt` などの動的な値を含むレスポンスでは、
そもそも型のみが検証可能です。

使用例

```go
AssertJSONResponseType[gen.HealthResponse](t, actual)
```

### `UseAppErrorHandler(t, e)`

本番相当の `HTTPErrorHandler` を Echo に登録します。素の `echo.New()` は Echo 標準の
エラーハンドラしか持たないため、`apperror` → HTTP ステータスの実マッピングを観測する異常系
テストでは、先に本番ハンドラを配線する必要があります。

各エンドポイントが返し得るエラーレスポンスは **OpenAPI の契約**（各オペレーションの
`responses`）で定義されます。異常系テストは、そのオペレーションが契約上宣言している
ステータスコードのみを対象とし、恣意的なコードは対象にしません。

### `AssertErrorResponse(t, actual, wantStatus)`

異常系レスポンスが `wantStatus` を返し、ボディが JSON のエラー形式（`ErrorResponse`）へ
デシリアライズ可能であることを検証します。`AssertJSONResponseType` と同様に、検証するのは
境界の関心事（`apperror` → ステータスのマッピングとエラーボディの形）のみで、`code` /
`message` の値の正しさはユニットテストの責務のままです。

使用例

```go
e := echo.New()
UseAppErrorHandler(t, e)
// ... usecase モックが apperror.ErrNotFound を返すよう設定し、handler を bind ...
actual := StartServer(t, e).DoJSON(http.MethodGet, path, nil, headers)
AssertErrorResponse(t, actual, http.StatusNotFound)
```

## Auth テストヘルパー

### `MakeAvailableUserID`

integration テストで **認証済みユーザーを模擬するヘルパー**です。

内部では Echo Middleware を追加し  
`ctxhelper.SetAuthnToEcho` を使って認証情報を設定します。

使用例

```go
headers := MakeAvailableUserID(t, e, userID)
srv.DoJSON(http.MethodPost, "/v1/users", body, headers)
```

## テスト設計ポリシー

integration テストでは以下の原則を守ります。

### 1 Infrastructure を使用しない

integration テストでは

- DB
- SQL
- Repository

を使用しません。

### 2 Usecase は mock する

integration テストでは **Usecase を mock 化**します。

理由: `integration = HTTP boundary test` であるためです。

### 3 HTTP を実際に叩く

handler を直接呼ぶのではなく `httptest.Server` を利用します。

### 4 レスポンス型で検証

レスポンスは **OpenAPI 型**で検証します。

- `gen.ResponseV1Users`
- `gen.ResponseHealth`
- `gen.ResponseVersion`
