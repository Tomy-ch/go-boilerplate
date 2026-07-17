# Outbox relay エンジンガイド（`internal/controller/outbox`）

[English](README.md) | 日本語

## オニオンアーキテクチャでの役割

- HTTP ハンドラや worker engine と同格の **poll 駆動 driving adapter**。新しいアーキテクチャ層ではなく、**Usecase 層へのもう 1 つの入口**。
- transactional outbox の **relay 側**：outbox store を周期 poll し、未 publish 行の配送を駆動する常駐 engine。対になる **emit 側**は呼び出し元の業務トランザクション内で同期実行され、usecase 層に属する。
- engine が担うのは **loop と待機制御だけ**。`claim → publish → mark` の業務は `outboxuc.RelayUsecase` に全委譲する。store・broker・トランザクションを直接触らない。
- 依存は usecase 層の port のみ：`outboxuc.RelayUsecase` / `clock.Sleeper` / `logging.Logger` / `observability.LayerTracer`（`TracerFactory.Controller()` 経由で取得）。`internal/infrastructure/*` は import しない（depguard `maintain_a_sound_controller` で機械担保）。

> relay が controller なのは、その責務が **cadence の orchestration**（poll 間隔・backoff・`ctx` 完了時の drain・span）であって業務ロジックではないから。`claim/publish/mark` のトランザクション、永続化 port、HTTP 送出 port はすべて usecase 境界の裏にある — worker engine が保つのと同じ分離。

## 公開 API

- `Engine` — 常駐 poll engine。`NewEngine(uc, sleeper, log, tf, set) *Engine` で結線し、`Run(ctx) error` が loop 本体。
- `Settings` — engine のチューニング値。DI 層が `OutboxConfig` から生成する：
  - `BatchSize int32` — 1 回の poll で claim する行数。
  - `PollInterval time.Duration` — 捌き切らなかったバッチ（空振り / 部分消化 / stall）の後に待機する時間。
  - `ErrorBackoff time.Duration` — `RelayBatch` がエラーを返した後に待機する時間。
  - **clamp（安全な既定値であり、silent ではない）：** `provideRelaySettings`（`internal/di/module/outboxrelay.go`）は `BatchSize` / `PollInterval` / `ErrorBackoff` が `0` / 負値に設定された場合、それぞれの既定値へ **clamp** する。非正の poll/backoff はスピン（ホットループ）してしまうため。これは失敗ではなく意図的な回復性の選択。`OUTBOX_*` env var は非ゼロの `envDefault` を持つため、clamp が発動するのは明示的な `0` 上書きのときだけ。silent にせずレビュー可能に保つため、ここに記し、セットアップレビュー（[`docs/get-started/setup-repository.md`](../../../docs/get-started/setup-repository.md)）にも列挙する。

## ループ意味論（`Run`）

`Run` は controller span を開始し、`ctx` 完了まで loop する（完了時は `nil` を返す）。各反復で `uc.RelayBatch(ctx, BatchSize)` を呼び、最大 `BatchSize` 件の pending 行を claim して `RelayResult`（`Claimed` / `Published`）を返す。反復後の待機判断：

| 結果 | 次の動作 | 理由 |
| --- | --- | --- |
| **満杯かつ進捗あり**（`Claimed >= BatchSize` かつ `Published > 0`） | **待機せず**即座に次 poll | まだ pending が残る可能性が高い。全速で捌き続ける |
| 空振り / 部分消化 / **満杯だが進捗ゼロ**（全件 publish 失敗） | `PollInterval` 待機 | 捌くものが無い、または下流が失敗している |
| `RelayBatch` エラー | ログ後 `ErrorBackoff` 待機 | 一時的な DB/broker 障害を落ち着かせる |

- **満杯だが進捗ゼロ → 必ず待機**のルールが load-bearing：全件失敗の満杯バッチを待機ゼロで再 claim すると、下流停止中にホットループして attempts を焼き切り、行を即座に `dead` 化する。stall した満杯バッチは常に待機へ落とす。
- 待機は `clock.Sleeper.Sleep(ctx, d)` 経由で行い、`ctx` キャンセルで即座に抜ける。
- `ctx` 完了は loop 先頭・`RelayBatch` エラー後・lag 記録前で再チェックし、shutdown 時に不要なエラーログや余分な RPC を出さない。

## Observability

- `observeLag` は outbox lag SLI（最古 pending 行の経過時間）を `uc.RecordLag(ctx)` でベストエフォート記録する：**バッチ成功時のみ**実行し（エラーバッチではスキップ。同一原因（DB 障害等）での二重ログを避けるため）、`ctx` 完了時はスキップ、失敗時はログのみでループを止めない。
- span は `Run` ごとに controller `LayerTracer` で 1 回開始する。engine は OpenTelemetry SDK を直接触らない。
- エラーログは `outbox-relay` logger 名の下で `logging.JobErrorKey` 構造化フィールドを使う。

## 配線とライフサイクル

- `OutboxRelayModule`（`internal/di/module/outboxrelay.go`）が提供し、**relay 専用プロセス**（`cmd outbox-relay`）でのみ使う。同 module は非標準の publisher HTTP クライアント profile（`MaxAttempts=1` 等）も閉じ込め、他プロセスへ漏れないようにする。
- `Settings` は `provideRelaySettings` が `OutboxConfig` から生成し、非正の `BatchSize` を `outboxuc.DefaultBatchSize` へ clamp してスピンループを防ぐ。
- `RegisterRelayHooks`（`internal/di/outboxrelay/hook`）が `SupervisedRunner` で loop を fx ライフサイクルに結線する：`OnStart` は `Run` を detached goroutine で起動（ブロックしない）、`OnStop` は engine の context をキャンセルし、stop 期限内で loop の終了を待つ。

## ファイル

- `relay.go` — `Engine`（`Run` / `waitDone` / `observeLag`）、`NewEngine`、`Settings`。adapter はこれで全て。broker 個別コードはここに無い（送出 port は usecase 境界の裏、実装は `internal/infrastructure/publisher`）。

## 関連

- Store 境界（relay が usecase 経由で駆動する永続化 port）: [`internal/usecase/boundary/outbox/README.ja.md`](../../usecase/boundary/outbox/README.ja.md)
- 詳細設計（役割論 / 状態遷移 / 実装箇所マップ / 用語集）: [docs/ja/design/outbox.ja.md](../../../docs/ja/design/outbox.ja.md)
