# integration ディレクトリ

このディレクトリは **結合テスト用 (integration test)** をまとめるための場所です。  

ユニットテストでは拾いきれない、Echo サーバを立ち上げて **実際の HTTP 通信経路**を通した検証を行います。

## integration ディレクトリがある理由

### 層の分離

ユニットテストは個々の関数やハンドラを対象としますが、結合テストは **ルータ〜ミドルウェア〜ハンドラ〜レスポンス整形までの一連の流れ**を確認します。  

このディレクトリを分けることで、テストの目的と粒度を明確にしています。

ユースケース層やリポジトリ層のテストはユニットテストで十分なため、ここでは扱いません。

### テストの粒度整理

`internal/controller/handler/...` → ハンドラ単位のユニットテスト  

`internal/integration` → Echo を実サーバとして立ち上げての結合テスト  

### 実運用に近い検証

実際にHTTP経由で叩いて確認するため、CI/CDのパイプラインやUAT/スモークテストの基盤としても利用可能です。

## 使いどころ

- **ハンドラ単位の分岐網羅** → ユニットテストで実施  
- **API が外部から正しく動作するか** → integration テストで実施  

これにより、テスト戦略として「詳細はユニットテスト」「経路全体は integration テスト」と役割分担ができます。

## integration_test.go で定義されている関数

### `StartServer(t *testing.T, e *echo.Echo) *Server`

Echo を `httptest.NewServer` で立ち上げ、結合テスト用の簡易サーバを返します。  

- 引数の `e` はハンドラがバインド済みの状態で渡します。
- `t.Cleanup` によりサーバは自動で停止します。
- 返される `Server` から `Do` / `DoJSON` を実行できます。

### `StopServer()`

明示的にサーバを停止します。  
通常は `StartServer` 内の `t.Cleanup` により自動で停止されるため、補助的に利用します。

### `Do(method, path string, reqBody io.Reader, contentType string, headers http.Header) *http.Response`

任意のメソッド・パス・リクエストボディ・ヘッダを指定して HTTP リクエストを送信します。  

- `http.NewRequestWithContext` を利用し、テストコンテキストを紐付けます。
- ヘッダや Content-Type の指定に対応しています。
- 実際の `http.Client` で通信し、レスポンスを返します。

### `DoJSON(method, path string, reqBody any, headers http.Header) *http.Response`

JSON 送受信用のショートカットです。  

- `reqBody` が指定されていれば JSON エンコードして送信します。
- `Content-Type: application/json` を自動付与します。
- 内部的に `Do` を呼び出しています。

### `AssertJSONResponse[T any](t *testing.T, _ T, actualResponse *http.Response)`

JSON レスポンスの内容を検証するユーティリティです。  

- ステータスコードが 200 OK であること。
- Content-Type が `application/json` を含むこと。
- レスポンスボディが指定した型 `T` に正しく Unmarshal できること。
- Unmarshal 結果が空でないこと。
