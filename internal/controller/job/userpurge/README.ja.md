# ユーザー物理削除ジョブガイド (`internal/controller/job/userpurge`)

[English](README.md) | 日本語

## オニオンアーキテクチャにおける役割

- **ワンショットの物理削除エントリポイント**（Controller 層 / CLI driving adapter）。新しいアーキテクチャ層ではなく、Usecase 層への入口の一つです。
- **退会の retention 側**です。退会そのものはユーザーを削除済みとして印を付ける（論理削除）だけで、保持期間を過ぎた行と従属データを最終的に消すのが本ジョブです。
- 外部スケジューラ（k8s CronJob / cron）が **デーモンではなく cron として**起動します。`cmd job user-purge` の 1 回の実行で削除して終了します。
- ジョブが担うのは **args のパース・span の開始/終了・結果ログ**のみで、削除の業務は `user.PurgeUsecase` に完全委譲します。Repository やトランザクションを直接触りません。
- サンプル User API の一部です。本パッケージは `make setup-remove-sample-api` で削除されます。

## 公開 API

- `New(logging logging.Logger, tf observability.TracerFactory, purge user.PurgeUsecase) job.Job` — DI コンストラクタ。`tf.Controller()` で Controller 層の tracer を取得します。`internal/di/module/job.go` の `group:"jobs"` に登録されます。
- `job.Job` インターフェース（`internal/usecase/boundary/job`）を実装します。
  - `Name() string` — ジョブキー `"user-purge"` を返します。
  - `Execute(ctx context.Context, args []string) error` — args をパースし、usecase へ委譲し、結果をログ出力します。

## 依存

| 依存 | 目的 |
| --- | --- |
| `user.PurgeUsecase` | `PurgeDeleted(ctx, retention, batchSize, dryRun) (PurgeResult, error)` — 退会から `retention` より長く経過したユーザーを `batchSize` 件ずつ物理削除し、削除件数とスキップ件数を返す |
| `logging.Logger` | 構造化された結果ログ |
| `observability.TracerFactory` | `tf.Controller()` による Controller 層 tracer |

## 実行セマンティクス (`Execute`)

1. Controller span を開始し（`tracer.Start`）、`defer` で終了します。
2. args を保持期間・バッチサイズ・dry-run フラグにパースし、`purge.PurgeDeleted(...)` を呼びます。
3. 成功時は削除件数を `logging.JobResultKey` に、スキップ件数を `logging.JobSkippedKey` に載せて **Info** でログ出力します。`--dry-run` ではメッセージが削除していないことを明示し、削除件数は「削除されるはずだった件数」になります。
4. 失敗時は、伝播する前に同じ 2 つの件数を **Warn** でログ出力します（理由は [job/README.ja.md § GC / バッチジョブ](../README.ja.md) を参照）。エラー自体はそのまま返します（Runner / CLI に伝播し、exit code は呼び出し側が決定します。ジョブは `os.Exit()` を呼びません）。

## Args

3 つの独立したフラグを、順不同で受け付けます。

| 入力 | 結果 |
| --- | --- |
| （なし） | 保持期間 `0` / バッチサイズ `0` → usecase 側の既定値が適用され、実削除になる |
| `--older-than=<duration>` | Go の duration 文字列（`720h` / `1h30m`）。usecase の既定は 30 日で、1 か月は `--older-than=720h` と書く（Go の duration に `d` 単位は無い） |
| `--batch-size=N`（正の int32） | `N` 件ずつ削除 |
| `--dry-run` | 削除せず件数のみ報告する |
| 未知のフラグ | エラー（削除しない） |
| いずれかのフラグの複数指定 | エラー |
| `--older-than` が解析不能 / `0` 以下 | エラー |
| `--batch-size` が非数値 / `0` 以下 | エラー |

## 補足

- 設計上べき等です。対象は経過時間の述語で決まるため、再実行しても既に条件を満たすユーザーを消すだけで、リトライは安全です。排他制御は持たず、多重起動の制御はスケジューラの関心事です（[ADR-0102 (scheduled-job-concurrency-delegated)](../../../../docs/adr/0102-scheduled-job-concurrency-delegated.md)）。
- 購入を保持しているユーザーは削除**されず**、`logging.JobSkippedKey` に件数として計上されます。その判断は usecase の関心事で、本ジョブは関与しません。
- 保持期間・バッチ処理・スキップ規則はいずれも usecase の関心事で、本ジョブは CLI の文字列を型付きの値へ変換するだけです。
