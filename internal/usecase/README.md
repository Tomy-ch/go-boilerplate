# ユースケース層（`internal/usecase`）ガイド

## オニオンアーキテクチャでの役割

- アプリケーションサービスとして、ユースケースの **手続き（ワークフロー）** を調整する層。
- 入力（DTO/VO）を受けて、**Domain（エンティティ/ドメインサービス）とRepository（ドメインの抽象）** を組み合わせ、結果（DTO）を返す。
- トランザクション境界と整合性の担保の単一起源（Tx開始/終了、リトライ方針など）。
- 外界（HTTP/DB/メッセージング）の詳細は知らない。純粋なアプリ語彙で完結する。

## この boilerplate での役割

- internal/usecase/<feature>/ に Command/Query のサービスを配置（例：user/）。
  - Command：作成/更新/削除（Txを開始し、Domainの不変条件を満たすように調整）。
  - Query（QS）：読み取り最適化。必要に応じて DTO で直接返す（Domainへはマップしない方針を許容）。
  - Pagination/Validation など プロトコル非依存のポリシーを一元化
    - 例：`NewPageFrom1Based`、`MaxPerPage`、`MaxOffsetAllowed`
- errorは `apperror.ErrXXX` にラップしてController層がHTTPにマップできるようにする。
- DI（fx）では Repository（interface/TxManager/Configなどの依存を受け取る。

## サードパーティを最小限に抑える

- Usecaseは原則 標準ライブラリのみ（context, time, errors, fmt など）。
- ORM・SQL実行・HTTPクライアント・EchoなどI/O系は一切持ち込まない。
- 型定義やDTOもプロジェクト内型で閉じる。sqlc生成型/driver型やOpenAPI生成型は上位/下位の層に隔離。
- テストも`testify`/`mock`程度に留め、モックはinterfaceベースで注入。
- どうしても必要な場合は、[pkg/](../../../pkg/)で薄いラッパーを作成する。

## 実装上の注意点

### 命名/構造

- インターフェイスは`Usecase`（例：`user.Usecase`）で統一。
- インスタンスの生成関数名は `New` で統一し、[di/usecase.go](../di/usecase.go) で登録する。

### ビジネスロジックを実装しない？ → 誤解を避けて明確化

- “ドメインロジック”は Domain 層に置く（エンティティ/VO/ドメインサービスのメソッド）。
- Usecase は“手続きロジック”（順序・Tx・外部境界の協調・入力検証と方針適用）を担当。

### HTTP/DBの要素は持ち込まない

- `http.*`, `echo.Context`, `sqlc` 型、`sql.Null*`、DB列名、OpenAPI生成型…を引数/戻り値に使わない。
- 代わりに DTO/VO（Page/Filters/Actor） で表現。

### エラー方針

- 入力の意味的な不正:
  - `apperror.ErrValidation` → 422
  - `apperror.ErrInvalidArgument` → 400
- 存在しない:
  - `apperror.ErrNotFound` → 404
- 競合:
  - `apperror.ErrConflict` → 409
- 一時的不可:
  - `apperror.ErrUnavailable` → 503
- 想定外:
  - そのまま or `apperror.ErrInternal` に包む → 500

### ページング

- NewPageFrom1Based(page, perPage) で既定値/上限/1→0変換を統一。
- Offset 上限（悪意対策）を超えたら `apperror.ErrInvalidArgument` を返す。

## 呼び出せる層 / 呼び出せない層

### 呼び出せる層

- Domain（エンティティ/ドメインサービス/Repositoryインタフェース）
- TxManager / Config / (必要なら) QueryService

### 呼び出せない層

- 他のUsecaseからの呼び出しは基本禁止（循環・肥大化を避ける。必要なら“アプリサービス（`Orchestrator`）”を別モジュールとして明示）。
- UsecaseからInfra/Controller/HTTP/OpenAPI/SQL"実装"は呼ばない。

## やっていいこと / いけないこと(まとめ)

### Do

- **DTO/VO（Page, Filters, Actor）**で受け渡し
- Tx 境界をここで定義（TxManager 経由で Do(ctx, func(txCtx){ ... })）
- Usecase層で初出の`apperror`でエラー分類を付与（errors.Is で Controller が判定しやすく）
- QSはRow→DTOに最短でマップ、CommandはDomainを介す。
- 表駆動テストでユースケースの分岐とTxの挙動を確認（testify）

### Don’t

- DomainのEntityを直接返す
- `http.Status` や `echo.Context` を引数に取る/返す
- `sqlc`生成型や`sql.Null*`を引数/戻り値に使う
- `openapi/gen`の型を直接返す（DTOに詰め替えるのはControllerの責務）
- Listで0件をエラー化（`apperror.ErrNotFound`(404)は単体取得のみ）
- 別のUsecaseを直接呼んで複雑化（必要なら`Orchestrator`を定義）

## Observability（Tracing）の使い方

この boilerplateUsecase層で直接OpenTelemetrySDKを扱わず、
observability.LayerTracerを経由してspanの開始・終了を行います。

### 1. Usecase層での span の開始と終了

各ハンドラーの先頭で必ず次の 2 行を記述してください。

```go
ctx, endSpan := s.tracer.Start(ctx)
defer endSpan()
```

- Start(ctx)でspanが開始され、trace_id/span_idがcontextに紐づきます。
- endSpan()は、spanの終了（span.End）を行います。
- defer endSpan()により例外や早期returnがあっても必ず終了されます。

ポイント：Usecase は span の開始・終了だけを知り、
OpenTelemetry SDK の詳細には一切触れません。

### 2. TracerのDI（observability.LayerTracer）

Usecaseは以下のようにobservability.LayerTracerを依存として受け取ります。

```go
type server struct {
    tracer   observability.LayerTracer
    txm      tx.Manager
    userRepo user.Repository // それぞれのリポジトリ
    pftRepo  prefecture.Repository
}
```

New関数内では、`observability.NewUsecaseTracer`でUsecase専用のトレーサーを生成します。

```go
func New(
    tf observability.TracerFactory,
    txm tx.Manager,
    userRepo user.Repository,
    prefectureRepo prefecture.Repository,
) Usecase {
    return &usecase{
        tracer:   tf.Usecase(),
        txm:      txm,
        userRepo: userRepo,
        pftRepo:  prefectureRepo,
    }
}
```

ここではSDKの生インスタンスを直接使わず、
observability層がtracerの生成ルール（レイヤー名やパッケージ名・関数名の抽出）を内部で隠蔽します。

## 最小スニペット（雛形）

```go
//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE -package=mock_$GOPACKAGE
// 唯一性のある名称
package user

import (
    "context"

    "boilerplate-go/internal/observability"
    // それぞれ実装で使うパッケージをimport
)


// 下位の層とやり取りするためのDTO
type UserMutableFields struct {
    FirstName      string
    LastName       string
    Email          string
    Phone          string
    PostalCode     string
    PrefectureName string
    City           string
    Street         string
    Building       *string
}

type CreateUserParamsDTO struct {
    UserID   uuid.UUID
    Password string

    UserMutableFields
}


// usecaseという名称は固定
type usecase struct {
    tracer   observability.LayerTracer
    txm      tx.Manager
    userRepo user.Repository
    pftRepo  prefecture.Repository
}

// Usecaseという名称は固定
type Usecase interface {
    GetAllUsers(ctx context.Context, page paging.Paging) ([]UserMutableFields, error)
    CreateUser(ctx context.Context, dto CreateUserParamsDTO) (UserMutableFields, error)

}

// Newという名称は固定
func New(
    tf observability.TracerFactory,
    txm tx.Manager,
    userRepo user.Repository,
    prefectureRepo prefecture.Repository,
) Usecase {
    return &usecase{
        tracer:   tf.Usecase(),
        txm:      txm,
        userRepo: userRepo,
        pftRepo:  prefectureRepo,
    }
}

func (u *usecase) GetAllUsers(ctx context.Context, page paging.Paging) ([]DTO, error) {
    // Spanの開始・終了呼び出して設定
    ctx, endSpan := u.tracer.Start(ctx)
    defer endSpan()

    // ユーザー一覧取得（Domainエンティティのスライス）
    us, err := u.userRepo.FindAll(ctx, page.Limit(), page.Offset())
    if err != nil {
       return nil, err
    }

    // オプション: observability.WithDomainSpanでDomain層のspanを作成
    // 可観測性を高めるために、Domain層の処理もspanとして切り出すことができます。
    // オプションなのでなくても構いません。
    // 第一引数のctxは、後続で使う場合は返り値を受け取って上書きしてください。
    ctx, prefectureMap, err := observability.WithDomainSpan(
        ctx, u.tracer, "user", "prefectureMap", func(ctx context.Context) (map[uuid.UUID]*prefecture.Entity, error) {
            // ユーザーの都道府県IDを集めて、一括で都道府県エンティティを取得
            pids := make([]uuid.UUID, len(us))
            for i, u := range us {
                pids[i] = u.PrefectureID()
            }

            // 都道府県エンティティの取得
            // IDsメソッドは複数IDで一括取得するリポジトリメソッドの例
            ps, pftErr := u.pftRepo.FindByIDs(ctx, pids)
            if pftErr != nil {
              return nil, pftErr
            }

            // 取得した都道府県エンティティをマップに詰め替え
            // Mapにすることで、後続のループで高速に参照できるようにする
            prefectureMap := make(map[uuid.UUID]*prefecture.Entity, len(ps))
            for _, p := range ps {
                prefectureMap[p.ID()] = p
            }

            return prefectureMap, nil
      })
    if err != nil {
      return nil, err
    }

    // ctxは、後続でobservability.WithDomainSpanを使わない場合は不要
    _, dtos, err := observability.WithDomainSpan(
        ctx, u.tracer, "user", "buildDTOs", func(ctx context.Context) ([]UserMutableFields, error) {
            // 結果をDTOに詰め替え
            dtos := make([]UserMutableFields, len(us))
            for i, u := range us {
                dtos[i] = UserMutableFields{
                    FirstName:  u.FirstName(),
                    LastName:   u.LastName(),
                    Email:      u.Email(),
                    Phone:      u.Phone(),
                    PostalCode: u.PostalCode(),
                    City:       u.City(),
                    Street:     u.Street(),
                    Building:   u.Building(),
                }
                // 都道府県名をマップから取得してセット
                if p, ok := prefectureMap[us[i].PrefectureID()]; ok {
                    dtos[i].PrefectureName = p.Name()
                }
            }
            return dtos, nil
        })

    return dtos, err
}

```
