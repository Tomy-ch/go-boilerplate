# Outbox GC ジョブガイド (`internal/controller/job/outboxgc`)

[English](README.md) | 日本語

## オニオンアーキテクチャにおける役割

- **ワンショットの GC エントリポイント**（Controller 層 / CLI driving adapter）。新しいアーキテクチャ層ではなく、Usecase 層への入口の一つです。
- [transactional outbox](../../../../docs/design/outbox.md) サブシステムの **prune 側**です。エントリは配信完了すると `published` にマークされます。本ジョブは retention を超えた `published` のエントリをバッチ削除し、outbox の無制限な肥大化を防ぎます。
- outbox の他の 2 つの非同期側とは別物です。relay のポーリングループ（`internal/controller/outbox` の常駐 `Engine`）でも、dead 行の復旧（`ReplayUsecase`）でもありません。本ジョブは `published` 行の刈り取りのみを行います。
- 外部スケジューラ（k8s CronJob / cron）が **デーモンではなく cron として**起動します。`cmd job outbox-gc` の 1 回の実行で掃除して終了します。
- ジョブが担うのは **args のパース・span の開始/終了・結果ログ**のみで、掃除の業務は `outbox.GCUsecase` に完全委譲します。store やトランザクションを直接触りません。

## 公開 API

- `New(logging logging.Logger, tf observability.TracerFactory, gc outbox.GCUsecase) job.Job` — DI コンストラクタ。`tf.Controller()` で Controller 層の tracer を取得します。`internal/di/module/job.go` の `group:"jobs"` に登録されます。
- `job.Job` インターフェース（`internal/usecase/boundary/job`）を実装します。
  - `Name() string` — ジョブキー `"outbox-gc"` を返します。
  - `Execute(ctx context.Context, args []string) error` — args をパースし、usecase へ委譲し、結果をログ出力します。

## 依存

| 依存 | 目的 |
| --- | --- |
| `outbox.GCUsecase` | `SweepPublished(ctx, batchSize) (int64, error)` — retention より古い `published` のエントリを `batchSize` 件ずつ削除し、合計削除件数を返す |
| `logging.Logger` | 構造化された結果ログ |
| `observability.TracerFactory` | `tf.Controller()` による Controller 層 tracer |

## 実行セマンティクス (`Execute`)

1. Controller span を開始し（`tracer.Start`）、`defer` で終了します。
2. args をバッチサイズにパースし、`gc.SweepPublished(ctx, batchSize)` を呼びます。
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

- 設計上べき等です。再実行しても対象の `published` のエントリを削除するだけなので、リトライは安全です。
- retention の幅は usecase の関心事で、本ジョブはバッチサイズのみを渡します。relay ループ・retention・dead のエントリの replay については[設計リファレンス](../../../../docs/design/outbox.md)を参照してください。
