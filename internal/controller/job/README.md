# コントローラー層（`internal/controller`）ガイド

## この boilerplate での役割

internal/controller/job は、CLI（Cobra）から起動される **バッチ/ジョブのエントリポイント（Controller層）**です。

この層の責務は、HTTP Handler と同じ思想で次に限定します。

- ジョブの入出力（引数 args の解釈、結果の出力形式）を整える
- Observability（LayerTracer）で span を開始・終了する
- args からジョブのビジネスロジックに必要なパラメーターを抽出・変換する
- ユースケース（Usecase層）を呼び出す
- エラーを apperror / errorresponse（または Job 用のエラー表現）に寄せる
- 結果をログに整形して出す

「ビジネスロジック」「DBアクセス」「ドメインモデルの操作」は Usecase / Domain / Infra に寄せ、Controller は薄く保ちます。

## 実装上の注意点

### 命名/構造

推奨の構造は「ジョブ 1 種類 = 1 パッケージ（1 ディレクトリ）」です。

命名は以下の方針が安定します。

- パッケージ名：lower_snake ではなく Go 流儀の lower（例：usercount, fixcollation）
- Job 名（Runner が引くキー）：kebab-case を推奨
  - 例：user-count / fix-collation / dump-schema
- Cobra の job <name> と一致させやすく、README にも書きやすい

## 呼び出せる層

- **Controller → Usecase のみ**（＋生成物`gen`、DTO/Presenter、`apperror`/`errorresponse`）。  
- **Infra / Domain を直接呼ばない**。  
- DI（`fx`）で `handler` は `usecase.Service` を受け取る。

## やっていいこと / いけないこと(まとめ)

### Do

- args を「そのジョブが必要とする引数」に 最小限に解釈する
  - 例：--dry-run、--limit、--since など
- 入力値のバリデーション（型変換・範囲チェック）を Controller で行う
  - ビジネスルール（例：状態遷移の可否）まではやらない
- Usecase を呼び出して結果を受け取り、ログや標準出力に整形して出す
- apperror を返す / 変換して返す（JobRunner 側で統一的に処理しやすい）
- LayerTracer で span を開始して、必ず defer で閉じる
- Job の開始/終了、入力、結果（件数など）を 構造化ログで残す

### Don’t

- DB ドライバや sqlc の Querier を Controller で直接触る（Infra の責務）
- Repository を Controller で直接呼ぶ（Usecase を飛ばさない）
- ドメインエンティティの生成・永続化ロジックを Controller に書く
- OTel SDK を直接触る（sdktrace.NewTracerProvider() 等を Controller で書かない）
- ジョブの途中で os.Exit() する（Runner/CLI の制御と衝突しやすい）
- 出力フォーマット（ログ/標準出力）の統一ルールを無視する（運用で地獄を見る）

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
type jobImpl struct {
    tracer  observability.LayerTracer // o11y用のトレーサー
    logging logging.Logger // 結果ログ出力用
    usecase hoge.Usecase // それぞれのジョブで使うユースケース
}
```

BindHandler側では、`observability.NewControllerTracer`でController専用のトレーサーを生成します。

```go
func New(
    tf observability.TracerFactory,
    usecase user.Usecase,
    logging logging.Logger,
) job.Job {
    return &jobImpl{
        tracer:  tf.Controller(),
        usecase: usecase,
        logging: logging,
    }
}
```

ここではSDKの生インスタンスを直接使わず、
observability層がtracerの生成ルール（レイヤー名やパッケージ名・関数名の抽出）を内部で隠蔽します。

## 参考スニペット

```go
package usercount

import (
    "context"

    "boilerplate-go/internal/observability"
    // それぞれ実装で使うパッケージをimport
)

// ジョブ名の定義
const jobName = "UserCountJob"

type jobImpl struct {
    tracer  observability.LayerTracer // o11y用のトレーサー
    usecase user.Usecase // コントローラからはUsecaseを呼び出す
    logging logging.Logger // 結果ログ出力用
}

// この関数をinternal/di/module/job.goで、[<package>.New,]として登録する。
func New(
    tf observability.TracerFactory,
    usecase user.Usecase,
    logging logging.Logger,
) job.Job {
    return &jobImpl{
        tracer:  tf.Controller(),
        usecase: usecase,
        logging: logging,
    }
}

// Name は、job.Job インターフェースの Name メソッドを実装します。
// 特に意図がなければこの実装をそのまま使ってください。
func (u *jobImpl) Name() string {
    return jobName
}

// Execute は、job.Job インターフェースの Execute メソッドを実装します。
func (u *jobImpl) Execute(ctx context.Context, args []string) error {
    // トレース用のspanを開始
    ctx, endSpan := u.tracer.Start(ctx)
    defer endSpan()

    // ジョブの主要ロジックをここに実装
    // 引数の解析
    var active *bool
    for _, a := range args {
        switch a {
        case "--active-only":
            active = ptr.To(true)
        case "--inactive-only":
            active = ptr.To(false)
        }
    }

    // ユースケースを呼び出し
    count, err := u.usecase.CountUsers(ctx, active)
    if err != nil {
        return err
    }

    // 結果をログに出力
    u.logging.Named(jobName).Info( // 出力はInfoレベル推奨 Nameでジョブ名を付与
        "Result: total user count",
        logging.Int64(logging.JobResultKey, count), // 結果の定数キーを使う
    )

    return nil
}
```
