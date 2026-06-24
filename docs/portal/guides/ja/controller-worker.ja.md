# Worker エンジンガイド（`internal/controller/worker`）

[English](README.md) | 日本語

## オニオンアーキテクチャでの役割

- HTTP ハンドラと同格の **message-in driving adapter**。新しいアーキテクチャ層ではなく、**Usecase 層へのもう 1 つの入口**。
- pull-ack キューを consume し、各メッセージを業務 `Handler` へ dispatch する。
- 依存は `internal/usecase/boundary/worker` の seam port（`Consumer` / `Handler` / `FailureHandler` / `Worker` / `State`）のみ。`internal/infrastructure/queue/*` は import しない（depguard `maintain_a_sound_controller` で機械担保）。

> port を（ここではなく）`internal/usecase/boundary/worker` に置くのは、層ルール上 engine（controller）と broker adapter（infrastructure）の双方が import できる唯一の場所だから。`job` が port を boundary に置くのと同じ理由。

## pull 型前提と第一想定プラットフォーム

- 本 worker は **pull 型**：consumer が `Receive` で**引く**。IF は **AWS SQS** と **GCP Pub/Sub（pull）** を第一想定に設計する。
- その他の pull-ack 系（Azure Service Bus / Cloudflare Queues HTTP pull など）は**事例**：基本は **adapter を書くだけ**で乗る（IF 変更不要）。
- **IF ごと書き換えが要るのは、pull-ack に本質的に合わない PF（push 配信・streaming-log）だけ**。
- **push 型（RabbitMQ 等）は対象外** — push 配信（Pub/Sub push・webhook 等）は HTTP controller の領分。理由：ワーカー領域では pull が多数派で、backpressure を consumer 側が握れる。

## 「止める」の 3 区別

混同しやすい。サーキットブレーカを **intake 側（引き続けるか）** に当てている点は、一般的な「下流呼び出しを守る」イメージと異なるため明記する。

| 機構 | 何を止めるか | 復帰 | プロセス |
| --- | --- | --- | --- |
| **Backoff** | 速度制御のみ（intake は止めない）。独立した状態ではなく Open の cooldown を `pkg/backoff` で指数的に伸ばす形で実現 | 自動 | 生存 |
| **Circuit Open** | 下流失敗の継続で **`Receive`（intake）を停止** | **自動**（Open→cooldown→Half-open→Closed） | 生存 |
| **Fatal** | drain して **engine を停止** | 手動（再起動） | 終了 |

- **Open↔Fatal の境界**：Retryable 失敗の継続はサーキットを段階的にエスカレート（Open→Half-open→Open ごとに cooldown 増分）。engine を落とす（Fatal）のは `Handler` が `apperror.ErrFatal` を返したとき（回復不能な設定不整合等）のみ。Circuit Open は一時的・自己回復、Fatal は終端。
- **Circuit（engine 全体）vs `Nack` 遅延（per-message）**：circuit は poll loop 全体を絞る（キューからどれだけ引くか）。per-message の再配送遅延は adapter の best-effort な `Nack` 挙動（SQS の可視性等）。両者は別レイヤで、`Nack` 遅延は **port 保証にしない**。broker 非依存の backpressure は circuit が担う。

## 不変条件（受け入れ基準）

engine は **in-memory fake**（`internal/usecase/boundary/worker/fake`）に対して完成し、実 broker 無しで全テストが green。テスト名は不変条件 ID A1–A7 / B1–B4 / C1（engine）＋ D1–D3（O11Y）にマップ。主なもの：

- A1/A2：成功時のみ `Ack`、Retryable で `Nack`。
- A5：Permanent → `FailureHandler` → `Ack`、Fatal → 停止。
- A6：1 メッセージの panic を recover し engine を巻き込まない。
- B1/B2/B3：同時数上限 / in-flight 上限 / `PartitionKey` 直列化。
- B4：サーキットブレーカ（Open で intake 停止、Half-open で回復）。
- C1：SIGTERM/SIGINT で in-flight を drain、未完は `Ack` しない（再配送）。
- D1–D3：traceparent 継続 / engine 所有 metric / 構造化ログ。

## ファイル

- `runner.go`（`Engine`：registry / `Run` / `Healthy`）、`run.go`（1 Run 単位の poll loop / dispatch / drain）。
- `circuit.go`（3 状態ブレーカ、cooldown は `pkg/backoff`）、`classify.go`（error→分類）、`settings.go`（engine-core `Settings`）、`dispatch.go`（`PartitionKey` 直列化）、`state.go`（`worker.State` 実装）、`errors.go`（registry sentinel）、`metrics.go` / `telemetry.go`（O11Y）。

SQS 参考 adapter（`internal/infrastructure/queue/sqs`）は **default では配線せず**、`aws-sdk-go-v2` を出荷バイナリに載せない。

> 詳細設計（状態遷移 / 実装箇所マップ / 用語集）: [docs/ja/design/worker.ja.md](../../../docs/ja/design/worker.ja.md)。
