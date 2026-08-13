# Idempotency GC ジョブガイド (`internal/controller/job/idempotencygc`)

[English](README.md) | 日本語

## オニオンアーキテクチャにおける役割

- **ワンショットの GC エントリポイント**（Controller 層 / CLI driving adapter）。新しいアーキテクチャ層ではなく、Usecase 層への入口の一つです。
- [idempotency](../../../../docs/design/idempotency.md) サブシステムの **housekeeping 側**です。リクエストパスは各冪等性キーのエントリに TTL を刻みます。本ジョブは TTL がすでに失効したエントリをバッチ削除し、ストアの無制限な肥大化を防ぎます。
- 外部スケジューラ（k8s CronJob / cron）が **デーモンではなく cron として**起動します。`cmd job idempotency-gc` の 1 回の実行で掃除して終了します。
- ジョブが担うのは **args のパース・span の開始/終了・結果ログ**のみで、`claim → sweep → count` の業務は `idempotency.GCUsecase` に完全委譲します。store やトランザクションを直接触りません。

## 公開 API

- `New(logging logging.Logger, tf observability.TracerFactory, gc idempotency.GCUsecase) job.Job` — DI コンストラクタ。`tf.Controller()` で Controller 層の tracer を取得します。`internal/di/module/job.go` の `group:"jobs"` に登録されます。
- `job.Job` インターフェース（`internal/usecase/boundary/job`）を実装します。
  - `Name() string` — ジョブキー `"idempotency-gc"` を返します。
  - `Execute(ctx context.Context, args []string) error` — args をパースし、usecase へ委譲し、結果をログ出力します。

## 依存

| 依存 | 目的 |
| --- | --- |
| `idempotency.GCUsecase` | `SweepExpired(ctx, batchSize) (int64, error)` — 失効したエントリを `batchSize` 件ずつ削除し、合計削除件数を返す |
| `logging.Logger` | 構造化された結果ログ |
| `observability.TracerFactory` | `tf.Controller()` による Controller 層 tracer |

## 実行セマンティクス (`Execute`)

1. Controller span を開始し（`tracer.Start`）、`defer` で終了します。
2. args をバッチサイズにパースし、`gc.SweepExpired(ctx, batchSize)` を呼びます。
3. 成功時は削除件数を `logging.JobResultKey` に載せて **Info** でログ出力します。
4. 失敗時は、伝播する前に削除件数を **Warn** でログ出力します（理由は [job/README.ja.md § GC / バッチジョブ](../README.ja.md) を参照）。エラー自体はそのまま返します（Runner / CLI に伝播し、exit code は呼び出し側が決定します。ジョブは `os.Exit()` を呼びません）。

## Args

`--batch-size=N` のみを受け付けます。

| 入力 | 結果 |
| --- | --- |
| （なし） | バッチサイズ `0` → usecase 側の既定値が適用される |
| `--batch-size=N`（正の int32） | `N` 件ずつ掃除 |
| 未知のフラグ | エラー（掃除しない） |
| `--batch-size` の複数指定 | エラー |
| `N <= 0` / 非数値 / 負数 | エラー |

## 補足

- 設計上べき等です。再実行しても失効済みのエントリを削除するだけなので、リトライは安全です。
- 本ドキュメントはこのジョブの役割のみを記述します。リクエストパスのオーケストレーション、`Store` シーム、そのインフラ実装は idempotency の usecase / infrastructure 層にあります。[設計リファレンス](../../../../docs/design/idempotency.md)を参照してください。
