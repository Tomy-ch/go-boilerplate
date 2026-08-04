# 商品画像 GC ジョブガイド (`internal/controller/job/productimagegc`)

[English](README.md) | 日本語

## オニオンアーキテクチャにおける役割

- **ワンショットの回収エントリポイント**（Controller 層 / CLI driving adapter）です。Usecase 層へのもう 1 つの入口であり、新しいアーキテクチャ層ではありません。
- 同期削除が届かない領域を担当します。DB トランザクションとストレージ削除は 2 相コミットできないため、トランザクション内で消すとロールバック時に画像だけが復旧不能になります。また「アップロードしたが商品を作らなかった」経路はドメインイベントが発生しないため、outbox 経由でも回収できません。ストレージ起点の照合は同期削除の代替ではなく、同期削除が原理的に届かない範囲を受け持ちます。
- 外部スケジューラ（k8s CronJob / cron）が **デーモンではなく cron** として起動する想定です。`cmd job product-image-gc` の 1 回の起動で掃引して終了します。
- ジョブが担うのは **args のパース・span の開始/終了・結果ログ**のみで、回収そのものは `product.ImageGCUsecase` に完全委譲します。ストレージや Repository を直接触りません。
- サンプル商品 API の一部です。このパッケージは `make setup-remove-sample-api` で削除されます。

## 公開 API

- `New(logging logging.Logger, tf observability.TracerFactory, gc product.ImageGCUsecase) job.Job` — DI コンストラクタ。`tf.Controller()` で Controller 層の tracer を取得します。`internal/di/module/job.go` の `group:"jobs"` に登録されます。
- `job.Job` インターフェース（`internal/usecase/boundary/job`）を実装します。
  - `Name() string` — ジョブキー `"product-image-gc"` を返します。
  - `Execute(ctx context.Context, args []string) error` — args をパースし、usecase へ委譲し、結果をログ出力します。

## 依存

| 依存 | 用途 |
| --- | --- |
| `product.ImageGCUsecase` | `SweepOrphans(ctx, grace, batchSize, dryRun) (ImageGCResult, error)` — `grace` より古く、どの商品からも参照されていないオブジェクトを列挙 1 ページ単位で削除し、削除件数と照合件数を返す |
| `logging.Logger` | 構造化された結果ログ |
| `observability.TracerFactory` | `tf.Controller()` による Controller 層 tracer |

## 実行セマンティクス (`Execute`)

1. Controller span を開始し（`tracer.Start`）、`defer` で終了します。
2. args を猶予期間・ページ件数・dry-run フラグにパースし、`gc.SweepOrphans(...)` を呼びます。
3. 成功時は削除件数を `logging.JobResultKey` に、照合件数を `logging.JobScannedKey` に載せて **Info** でログ出力します。`--dry-run` ではメッセージが削除していないことを明示し、削除件数は「回収されるはずだった件数」になります。
4. 失敗時は、伝播する前に同じ 2 つの件数を **Warn** でログ出力します。削除済みのオブジェクトは復元できないため、件数を捨てると既に消えた画像が見えなくなります。エラー自体はそのまま返します（Runner / CLI に伝播し、exit code は呼び出し側が決定します。ジョブは `os.Exit()` を呼びません）。

## Args

3 つの独立したフラグを、順不同で受け付けます。

| 入力 | 結果 |
| --- | --- |
| （なし） | 猶予期間 `0` / ページ件数 `0` → usecase 側の既定値が適用され、実削除になります |
| `--older-than=<duration>` | Go の duration 文字列（`48h`、`1h30m`）。usecase の既定は 24 時間で、Go の duration に `d` 単位は存在しません |
| `--batch-size=N`（正の int32） | 1 ページ `N` 件で列挙します。照合と削除も同じページ単位です |
| `--dry-run` | 削除せず件数のみ報告します |
| 未知のフラグ | エラー（何も回収しません） |
| 同一フラグの複数回指定 | エラー |
| `--older-than` が解釈不能 / `<= 0` | エラー |
| `--batch-size` が非数値 / `<= 0` | エラー |

## 補足

- **猶予期間がこの方式の核心です。** アップロード直後のオブジェクトは、商品作成フォームを記入中のものと区別がつきません。年齢述語なしでは正常なアップロードを削除してしまいます。
- **参照照合の失敗が削除へ流れ落ちることはありません。** 「照合に失敗した」を「参照されていない」と倒すとバケット内の全画像を消すことになります。これが唯一の致命的な失敗モードなので、エラーはそのページの削除を行わずに中断します。
- 対象は `products/` 接頭辞のキーだけです。列挙後にも接頭辞を再検査するため、絞り込みを無視するストレージ実装に当たっても無関係なオブジェクトが削除されることはありません。
- 設計上冪等です。対象集合は年齢述語と参照照合で決まるため、再実行しても既に条件を満たすものだけが回収され、リトライは安全です。排他制御は持たず、並行制御はスケジューラの責務です（[ADR-0095](../../../../docs/adr/0095-scheduled-job-concurrency-delegated.md)）。
- DB 台帳方式（`product_images` テーブル + tombstone）は精確でバケット走査も不要ですが、migration とアップロード経路への書き込み追加を要します。スケール時の答えであって、出発点ではありません。
