# コントローラー層（`internal/controller`）ガイド

## オニオンアーキテクチャでの役割

- **外界（HTTP/REST）とアプリケーションの境界面**。
- **プロトコル適応（Adapter）** を担い、入力を **アプリ語彙（DTO/Value）** に変換して **Usecase** を呼ぶ。
- **出力整形（Presenter）** でUsecaseの結果を **OpenAPI のレスポンス型** へ詰め替える。
- 例外（`error`）を **HTTP ステータス** ＋ **エラーコード** へマッピング（`apperror` → Status）。

> ポイント：**ビジネスロジックは一切持たない**。持つのは「HTTPの解釈と整形」だけ。

## この boilerplate での役割

- Echo のハンドラ（`handler/v1/...`）で **OpenAPI 生成型（`gen`）の ServerInterface** を実装。
  1. 入力のパース/軽い検証（型・必須チェック）
  2. Usecase呼び出し
     - 必要なら送信用の **DTO/VO** に詰め替えて渡す
  3. Usecaseから返却されたDTO→OpenAPI型への詰め替え
- エラーは[apperrorで定義されているマップピング](../apperror/README.md)で apperrorのエラーを統一マッピングして返却される。
- ページングは`paging.NewPageFrom1Based()` に渡して正規化。
- リクエストID/ロギングなどはミドルウェア（Echo + Zap）で実施。

## oapi-codegenからハンドラの生成

- 生成する際には、[openapi/openapi.yaml](../../openapi/openapi.yaml)に定義したルーティングに沿う形で、
  `internal/controller/handler/`の先のディレクトリをURIとして再現してハンドラファイルを生成してください。
  1. [openapi/openapi.yaml](../../openapi/openapi.yaml)などにAPI定義を作成
     - [生成を前提としたOpenAPIガイドライン](../../openapi/README.md)を参照
  2. `internal/controller/handler/`の先のディレクトリをURIとして再現
     - 例1: `/v1/users` → `internal/controller/handler/v1/users/`
     - 例2: `/v1/users/{id}` → `internal/controller/handler/v1/users/detail/`
  3. 作成したファイルの先頭に生成用のコメントを追加

     ```go
     //go:generate oapi-codegen --include-tags=v1/users --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
     //go:generate oapi-codegen --include-tags=v1/users --package=gen --generate=echo-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml
     ```

  4. `swagger-cli`でOpenAPIを結合・検証し、`oapi-codegen`で生成
     - `make gen` で一括生成可能
  5. `internal/controller/handler/<version>/<resource>/gen`に生成物が出力される。
  6. 生成物をもとにコントローラーを実装する。
  7. 実装したコントローラーの`BindHandler`を[コントローラー層のDIモジュール](../di/handler.go)に登録する。
- 入出力の型（パラメータ・ボディ・レスポンス）は生成物を使用し、**Usecase の DTO と明確に分離**。  
- スキーマ変更は **OpenAPI → 再生成 → 実装調整**の一方向。生成物は**編集禁止**。

## 実装上の注意点（HTTPの要素をUsecaseに漏らさない）

### 命名/構造

- ルーティングの登録関数は `BindHandler` とし、[di/handler.go](../di/handler.go)で登録する。
- URIとした時のリソース構造を再現し、パスパラメータに関しては`detail`で分離して命名する。
  - 例1: package v1users
  - 例2: package v1usersdetail

### UsecaseにHTTPの語彙を入れない

- `http.Request`/`http.Header`/`http.Status*`などの`http.`を**絶対**渡さない。  
- Usecaseの引数はDTO / VO（例：`Page`）/ Contextのみ。

### ページング

- Controller: `page & per_page`を受け取り、`usecase.NewPagingFrom1Based()`でhttpを意味（Paging）へ変換。
- Usecase: `Paging`を受け、方針（上限・既定）を一元管理。

### エラーマッピング

- Controller層で直接定義し、呼び出される基底のエラー（`apperror`）は下記のもの。
  - `ErrInvalidArgument` → 400
  - `ErrUnauthenticated` → 401
  - `ErrUnimplemented` → 501
  - `ErrUnavailable` → 503
- 0件リストは**正常**（200 + 空配列）。**NotFound は単体取得のみ**。  

### トランザクション

- Controller は Tx を知らない。Tx 境界は Usecase（`TxManager`）が握る。

## 呼び出せる層

- **Controller → Usecase のみ**（＋生成物`gen`、DTO/Presenter、`apperror`/`errorresponse`）。  
- **Infra / Domain を直接呼ばない**。  
- DI（`fx`）で `handler` は `usecase.Service` を受け取る。

## やっていいこと / いけないこと(まとめ)

### Do

- `Get...Params` → **VO/DTO（Page, Filters など）**へ変換
- DTO → `gen` レスポンスへ **Presenter** として詰め替え
- `httptest` + `testify` で **エンドツーエンド風** にハンドラを検証

### Don’t

- Usecase に `http.Status`, `echo.Context`, `*http.Request` などの HTTP 要素を渡す
- Usecase で `limit/offset` を直に決めるために **HTTP のパラメータ生値**を渡す  
- `sqlc` 生成型やDB列名をControllerにそのまま持ち込む
- 一覧0件で `ErrNotFound` を返して404にする  
- 自動判定しているステータスコードを変更するために、エラーを握りつぶして独自に返す
- ログは **Zap ミドルウェア**（リクエストID, ルート, ステータス, 所要時間）

## Observability（Tracing）の使い方

このboilerplateでは、Controller層で直接OpenTelemetrySDKを扱わず、
observability.LayerTracerを経由してspanの開始・終了を行います。

### 1. Controller層でのspanの開始と終了

各ハンドラーの先頭で必ず次の2行を記述してください。

```go
ctx, endSpan := s.tracer.Start(ctx)
defer endSpan()
```

- Start(ctx)でspanが開始され、trace_id/span_idがcontextに紐づきます。
- endSpan()は、spanの終了（span.End）を行います。
- defer endSpan()により例外や早期returnがあっても必ず終了されます。

ポイント：Controllerはspanの開始・終了だけを知り、
OpenTelemetry SDK の詳細には一切触れません。

### 2. TracerのDI（observability.LayerTracer）

Controllerは以下のようにobservability.LayerTracerを依存として受け取ります。

```go
type server struct {
    tracer observability.LayerTracer
    uc      user.Usecase // それぞれのユースケース
}
```

BindHandler側ではDIコンテナから渡されたtrace.TracerProviderとzap.Loggerを用いて、
`observability.NewControllerTracer`でController専用のトレーサーを生成します。

```go
func BindHandler(
  e *echo.Echo, tf observability.TracerFactory, uc user.Usecase
) {
    gen.RegisterHandlers(e, gen.NewStrictHandler(&server{
        tracer: tf.Controller(),
        uc:     uc,
    }, nil))
}
```

ここではSDKの生インスタンスを直接使わず、
observability層がtracerの生成ルール（レイヤー名やパッケージ名・関数名の抽出）を内部で隠蔽します。

## 参考スニペット

```go
//go:generate oapi-codegen --include-tags=v1/users --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/users --package=gen --generate=echo-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// パッケージ名は衝突防止のためURIに合わせてください
package v1users

import (
    "context"

    "boilerplate-go/internal/observability"
    // それぞれ実装で使うパッケージをimport

    "github.com/labstack/echo/v4"
)


type server struct {
    tracer observability.LayerTracer
    uc      user.Service
}

// この関数をdi/handler.goで、[<package>.BindHandler,]として登録する。
func BindHandler(
  e *echo.Echo, tf observability.TracerFactory, uc user.Service,
) {
    gen.RegisterHandlers(e, gen.NewStrictHandler(&server{
        tracer: tf.Controller(),
        uc: uc,
    }, nil))
}

// handler
func (s *server) GetV1UsersDetail(ctx context.Context, request gen.GetUsersRequestObject) (gen.GetUsersResponseObject, error) {
    // Spanの開始・終了呼び出して設定
    ctx, endSpan := s.tracer.Start(ctx)
    defer endSpan()

    page := usecase.NewPageFrom1Based(request.Params.Page, request.Params.PerPage)

    // Usecase 呼び出し（DTO返却）
    // user.ConditionByName はUsecaseの持ち物
    list, err := s.uc.GetV1UsersByName(ctx, user.ConditionByName{
        NameKeyword: ptr.StringVal(request.Params.NameKeyword),
        Page:        page,
    })
    if err != nil {
        // エラーの基底値に従って、対応するHTTPステータスを返すのでハンドリングは不要
        return err
    }

    // プレゼンター処理(DTO → OpenAPIの型)
    users := make([]gen.UserResponse, len(dtos))
    for i, dto := range dtos {
      users[i] = gen.UserResponse{
        Name:  dto.Name,
        Email: types.Email(dto.Email),
        Phone: ptr.To(dto.Phone),
      }
    }

    res := gen.ResponseV1Users{
      Users:  users,
      Limit:  page.Limit(),
      Offset: page.Offset(),
    }

    return gen.GetUsersByName200JSONResponse(res), nil
}
```
