# handlertest（`echotest.go` / `assert.go`）README

Echo を使った **コントローラ層テスト**を “最短＆安全” に書くための小さなユーティリティです。  

このディレクトリには次の 2 ファイルがあります。

- `echotest.go` … **HTTP リクエスト生成＋実行**を支援するビルダー
- `assert.go` … **JSON 等価**や **Echo ルート検証**のためのアサート関数

以下では、 **ユニット（UT）** と **結合/UAT 風**の両スタイルでの使い方をまとめます。

## 前提

- Echo v4
- Go ≥ 1.18（`AssertJSONEqual[T]` のジェネリクス使用）
- 生成ハンドラやルータ登録は各プロジェクトの `BindHandler(e, ...)` 等に合わせてください

## 1 EchoTestClient の基本（`echotest.go`）

`EchoTestClient` は、**テスト用の HTTP リクエストを組み立てるビルダー**です。  
用途に応じて 2 つの使い方があります。

### A. ルータを通して実行（結合/UAT寄り）

- `RequestURL("/path?query=...")` を指定 → **Echo のルータが path param を自動解決**（内部で `Router().Find`）
- `Serve()` を使うと **`e.ServeHTTP` まで一発**で実行

```go
e := echo.New()
BindHandler(e /*, 依存があれば渡す */)

rec := handlertest.
  NewEchoTestClient(t, e).
  Method(http.MethodGet).
  RequestURL("/health").
  Serve()

require.Equal(t, http.StatusOK, rec.Code)
```

### B. ハンドラを直叩き（UT寄り）

- ルータ依存を避けたいときは、**ルート定義**＋**パスパラメータ**を手動で設定
- `Build()` が `req/rec/ctx` を返すので、**ハンドラ関数を直接呼ぶ**

```go
e := echo.New()

_, rec, ctx := handlertest.
  NewEchoTestClient(t, e).
  Method(http.MethodGet).
  RoutePattern("/v1/users/:id").
  PathParams([]handlertest.EchoTestParam{{Name: "id", Value: "123"}}).
  Build()

s := &server{/* 依存注入（mock等） */}
require.NoError(t, s.GetUser(ctx))
require.Equal(t, http.StatusOK, rec.Code)
```

> どちらの方法でも `JSONBody`, `RawBody`, `Header`, `AuthBearer`, `QueryParams` が使えます。

## 2 よく使う組み合わせ（レシピ）

### GET（PathParam + Query）

```go
rec := handlertest.
  NewEchoTestClient(t, e).
  Method(http.MethodGet).
  RequestURL("/v1/users/123").                    // ルータに解決させる
  QueryParams([]handlertest.EchoTestParam{
    {Name: "verbose", Value: "1"},
  }).
  Serve()

require.Equal(t, http.StatusOK, rec.Code)
```

### POST（JSON）

```go
_, rec, ctx := handlertest.
  NewEchoTestClient(t, e).
  Method(http.MethodPost).
  RoutePattern("/v1/users").
  JSONBody(gen.CreateReq{Name: "Taro"}).
  Build()

// ハンドラ直叩き（UT）
require.NoError(t, s.CreateUser(ctx))
require.Equal(t, http.StatusCreated, rec.Code)
```

## 3 JSON の検証（`assert.go`）

### `AssertJSONEqual[T]`

HTTP ステータスと **JSON ボディの完全一致**（型＋値）をまとめて検証します。

```go
expected := gen.ResponseHealth{Status: "ok"}

// ルータ経由で実行
rec := handlertest.
  NewEchoTestClient(t, e).
  Method(http.MethodGet).
  RequestURL("/health").
  Serve()

handlertest.AssertJSONEqual(t, http.StatusOK, expected, rec)
```

## 4 Echo のルート検証（`assert.go`）

### HTTP メソッドの検証

```go
handlertest.AssertEchoRouterMethods(t,
  []string{http.MethodGet},
  e.Routes(),
)
```

### パスの検証

```go
handlertest.AssertEchoRouterPath(t, "/health", e.Routes())
```

## 5 並列実行（`t.Parallel()`）時の注意

- **Echo インスタンスはサブテストごとに新規作成**してください
  - `e := echo.New()` を親で共有し、並列で `BindHandler`/`Router().Find` するとレースの原因になります
- **gomock** は **各サブテスト内で `NewController(t)` → `defer Finish()`** の形なら安全
- 読み取り専用の期待値（DTO 等）は共有で OK

## 6 どのテストで何を見るか（役割分担のおすすめ）

- **UT（ハンドラ直叩き）**  
  - 分岐網羅（bind/変換/ユースケース呼び出し/エラー種別→HTTP 変換/レスポンス整形）  
  - `RoutePattern + PathParams + Build()` → `s.Method(ctx)`
- **結合（ルータ経由）**  
  - 代表ケースを `RequestURL(...).Serve()` で 1〜2 本  
- **UAT スモーク**  
  - `httptest.NewServer(e)` or `Serve()` で **200 だけ**、必要なら **型に Unmarshal が通るか**まで

## 7 API 一覧（抜粋）

### EchoTestClient

- `NewEchoTestClient(t, e *echo.Echo)`  
- `Method(string)` / `Header(k, v string)` / `AuthBearer(token string)`
- `RoutePattern(string)` / `PathParams([]EchoTestParam)`  
- `RequestURL(string)` / `QueryParams([]EchoTestParam)`
- `JSONBody(any)` / `RawBody(io.Reader, contentType string)`
- `Build() (*http.Request, *httptest.ResponseRecorder, echo.Context)`  
- `Serve() *httptest.ResponseRecorder`（`e.ServeHTTP` を実行）

### assert

- `AssertJSONEqual[T any](t, expectedCode int, expected T, rec *httptest.ResponseRecorder)`
- `AssertEchoRouterMethods(t, expected []string, routes []*echo.Route)`
- `AssertEchoRouterPath(t, expected string, routes []*echo.Route)`

## 8 Tips

- **Query を個別に積みたい**場合は `QueryParams` を使う（`RequestURL` に直接 `?` を書くより typo を防げます）
- **`Serve()` は内部で `e.ServeHTTP` を直叩き**（外にサーバを立てません）。本当に HTTP レイヤを通したいなら `httptest.NewServer(e)` を使う別テストを。
- **型だけ担保する UAT**は `var out gen.Type; json.Unmarshal(body, &out)` のパターンが軽くて壊れにくい。
