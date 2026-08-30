# Orphan Cleanup ジョブガイド（`internal/controller/job/orphancleanup`）

## オニオンアーキテクチャにおける役割

- **ワンショットの回収エントリポイント**（Controller 層 / CLI 駆動アダプタ）。Usecase 層へのもう 1 つの入口であって、新しいアーキテクチャ層ではありません。
- [Realtime Delivery](../../../../docs/design/realtime-delivery.ja.md) の fan-out における復旧側の半分です。serve instance は起動時に自分専用の SQS queue と SNS subscription を作り、停止時に削除します。その停止を経ずに死んだ instance は両方を残し、それらを指し示す索引は instance lease だけです。本ジョブは、その lease が名指すものを回収します。
- 外部スケジューラ（k8s CronJob / cron）が**デーモンではなく cron として**起動します — `cmd job orphan-cleanup` の 1 回の呼び出しが掃除して終了します。アプリケーション内部からは起動されず、アプリ内の singleton 制御も置きません（[ADR-0109](../../../../docs/adr/0109-scheduled-job-concurrency-delegated.ja.md)）。引き受けを条件付き書き込みで取るため、2 つが同時に走っても安全だからです。
- 本ジョブが持つのは**引数の拒否・スパンの開始と終了・結果ログ**だけです。どの instance を回収してよいか、どの順序で行うかは `realtime.OrphanSweeper` に完全に委譲します。store や fan-out を直接触ることはありません。

## 公開 API

- `New(logging logging.Logger, tf observability.TracerFactory, newSweeper SweeperFactory) job.Job` — DI コンストラクタ。Controller 層のトレーサを `tf.Controller()` で取得します。`group:"jobs"` への登録は `JobModule()` ではなく `internal/di/module/realtimecleanup.go` が行うため、共有のジョブモジュールは Realtime への依存を持ちません（[`internal/di/module/README.ja.md`](../../../di/module/README.ja.md) 参照）。
- `SweeperFactory` — `func(ctx) (realtime.OrphanSweeper, error)`。掃除役そのものではなくファクトリを受け取るのは、fx が `Runner` を組むために登録済みジョブの constructor をすべて実行するためです。掃除役を graph に載せると、Realtime を設定していない環境で無関係なジョブ（`outbox-gc` ほか）まで起動できなくなります。ここで組み立てれば、`REALTIME_TOPIC` の欠落はこのジョブが実行されたときにこのジョブだけを失敗させます。
- `job.Job` インターフェース（`internal/usecase/boundary/job`）を実装します:
  - `Name() string` — ジョブキー `"orphan-cleanup"` を返します。
  - `Execute(ctx context.Context, args []string) error` — 引数を拒否し、usecase へ委譲し、結果をログに出します。

## 依存

| 依存 | 目的 |
| --- | --- |
| `realtime.OrphanSweeper` | `Sweep(ctx) (SweepResult, error)` — 回収できる lease を引き受け、それが名指す受信先を片付け、そのうえで lease を閉じる |
| `logging.Logger` | 構造化された結果ログ |
| `observability.TracerFactory` | `tf.Controller()` による Controller 層トレーサ |

## 実行の意味論（`Execute`）

1. Controller スパンを開始（`tracer.Start`）し、終了を `defer` する。
2. 引数があれば拒否し、`sweeper.Sweep(ctx)` を呼ぶ。
3. 成功時は内訳を **Info** で出す。
4. 一部が失敗した場合は内訳を **Warn** で出してから伝播する（理由は [job/README.ja.md § GC / バッチジョブ](../README.ja.md)）。エラーはそのまま返し、終了コードは Runner / CLI が決めます。ジョブが `os.Exit()` を呼ぶことはありません。

## 引数

ありません。引数があればエラーで、何も掃除しません。掃除に調整の余地はなく、引数を黙って無視するとスケジューラの設定ミスを隠してしまうためです。

## 結果の内訳

Realtime 固有のキーではなくジョブ共通のログキーで報告します。内訳はジョブの成果であり、兄弟ジョブと同じ読み方になるためです。

| 内訳 | キー | 意味 |
| --- | --- | --- |
| 検出 | `logging.JobScannedKey` | 期限切れが cleanup の安全余裕を過ぎている lease |
| 回収 | `logging.JobResultKey` | 受信先を片付け、lease まで閉じられたもの |
| 見送り | `logging.JobSkippedKey` | 他の掃除役が引き受けていた、または lease を閉じる前に instance が復帰した |

instance ごとの失敗件数はログに載せません。返すエラーが原因を 1 件ずつ包んで運んでおり、ここで数えるとエラーチェーンの言い直しになるためです。これらの内訳を metric にするのは可観測性のフェーズの仕事で、ここではありません。

## 補足

- 設計上冪等です。2 回目の実行は引き受けるものを見つけないので、再試行やスケジュールの重なりは安全です。
- 時間に関する値（期限・cleanup の安全余裕・引き受けの有効期限）は `internal/usecase/realtime` のものです。本ジョブはそのいずれも知りません。
