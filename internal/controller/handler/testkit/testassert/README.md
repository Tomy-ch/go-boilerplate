# `testassert` パッケージ

Echo を使った **コントローラ層テスト**を “最短＆安全” に書くための小さなユーティリティです。  

このディレクトリには次のファイルがあります。

- `assert.go` … **JSON 等価**や **Echo ルート検証**のためのアサート関数

## 前提

- Echo v4
- Go ≥ 1.18（`AssertJSONEqual[T]` のジェネリクス使用）
- 生成ハンドラやルータ登録は各プロジェクトの `BindHandler(e, ...)` 等に合わせてください

## 使用想定（アサート中心）

このディレクトリは主に「レスポンスの JSON 等価検証」や「ルータ定義のアサーション」を提供します。ハンドラへの入力生成や HTTP 層の実行はプロジェクト側のテストヘルパーに委ね、ここでは結果検証に集中してください。

## JSON の検証（`assert.go`）

### `AssertJSONEqual[T]`

HTTP ステータスと **JSON ボディの完全一致**（型＋値）をまとめて検証します。テスト内で HTTP レスポンスを得た後、期待値となる構造体と比較してください。

## Echo のルート検証（`assert.go`）

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

## 並列実行（`t.Parallel()`）時の注意

- **Echo インスタンスはサブテストごとに新規作成**してください
  - `e := echo.New()` を親で共有し、並列で `BindHandler`/`Router().Find` するとレースの原因になります
- **gomock** は **各サブテスト内で `NewController(t)` → `defer Finish()`** の形なら安全
- 読み取り専用の期待値（DTO 等）は共有で OK

## どのテストで何を見るか（役割分担のおすすめ）

- **UT（ハンドラ直叩き）**  
  - 分岐網羅（bind/変換/ユースケース呼び出し/エラー種別→HTTP 変換/レスポンス整形）  
  - `RoutePattern + PathParams + Build()` → `s.Method(ctx)`
- **結合（ルータ経由）**  
  - 代表ケースを `RequestURL(...).Serve()` で 1〜2 本  
- **UAT スモーク**  
  - `httptest.NewServer(e)` or `Serve()` で **200 だけ**、必要なら **型に Unmarshal が通るか**まで

## API 一覧（抜粋）

### assert

- `AssertJSONEqual[T any](t, expectedCode int, expected T, rec *httptest.ResponseRecorder)`
- `AssertEchoRouterMethods(t, expected []string, routes []*echo.Route)`
- `AssertEchoRouterPath(t, expected string, routes []*echo.Route)`

## Tips

- **Query を個別に積みたい**場合は `QueryParams` を使う（`RequestURL` に直接 `?` を書くより typo を防げます）
-- 本当に HTTP レイヤを通したいなら `httptest.NewServer(e)` を使う別テストを。
- **型だけ担保する UAT**は `var out gen.Type; json.Unmarshal(body, &out)` のパターンが軽くて壊れにくい。
