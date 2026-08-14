---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [outbox, async, concurrency]
---

# ADR-0054: SELECT FOR UPDATE SKIP LOCKED を使った単一トランザクションリレー（複数インスタンス間で安全）

English canonical: [0054-skip-locked-outbox-relay.md](0054-skip-locked-outbox-relay.md)

## ステータス

accepted

## 背景

可用性やスループットのスケーリングのために、複数のリレーインスタンスが同時に動作することがある。行レベルの調整なしでは、2 つのインスタンスが同じ `pending` 行をクレームし、それぞれがレシーバーエンドポイントにデリバリーしてしまい、重複を最小化するという目標に反する。ロック戦略は、各行を同時に 1 つのインスタンスのみが処理することを保証しつつ、他のインスタンスが異なる行で処理を続けられるようにしなければならない。

## 決定

`ClaimPending` は `outbox` テーブルに対して `SELECT ... FOR UPDATE SKIP LOCKED` を発行する。**クレーム → 発行 → マーク**の全シーケンスが単一のトランザクション内で実行される。クレームされた行のロックは、周囲のトランザクションがコミットまたはロールバックするまで保持される。そのため、並行して `ClaimPending` を実行する 2 つ目のリレーインスタンスは、ロックされた行でブロックするのではなくスキップする。

SQL は以下の通り:

```sql
SELECT id, message_id, aggregate_type, aggregate_id,
       event_type, payload, headers, attempts
FROM outbox
WHERE status = 'pending'
ORDER BY id
LIMIT $1
FOR UPDATE SKIP LOCKED;
```

## 影響

### ポジティブな影響

- マルチインスタンスの安全性が、アプリケーション側の調整や分散ロックマネージャーなしに、データベースレベルで担保される。
- `SKIP LOCKED` により行頭ブロッキングを回避できる。あるインスタンスが行を保持していても、他のインスタンスはその行をスキップして他の行の処理を続けられる。
- 各バッチ内では行が `id` 順に処理される（ベストエフォート。ロックの可用性のために厳密なグローバル順序は犠牲にする）。

### ネガティブな影響

- `FOR UPDATE SKIP LOCKED` をサポートするデータベース（PostgreSQL 9.5+）が必要。他のエンジンへの移植性は保証されない。
- 多数の行ロックを保持する長いリレートランザクションは競合を増加させる可能性がある。ロック保持時間を制限するためにバッチサイズを調整しなければならない。

## 検討した代替案

### ステータスに対する楽観的比較・セット

処理前に `UPDATE outbox SET status = 'claimed' WHERE status = 'pending'` を実行する。ロックを避けられるが、適切な分離レベルなしに高競合時に競合状態のリスクがあり、障害/クラッシュシナリオで処理しなければならない `claimed` ステータスが別途追加される。

### アプリケーションレベルのアドバイザリーロック

アプリケーションが管理する PostgreSQL アドバイザリーロック。より柔軟だが、正確に実装するには複雑で、どの行がロックされているかのアプリケーション側調整が必要。

### 排他的リレー（シングルインスタンス）

シンプルで並行性の問題がない。しかしデリバリーパス全体の単一障害点を生む。

## 補足

- マルチインスタンス安全性の不変条件: `docs/design/outbox.md`（§「Design invariants」）。
- SQL ソース: `database/dml/system_cqrs/outbox/claim_pending_outbox.sql`。
- 関連 ADR: [ADR-0052](0052-transactional-outbox.ja.md)、[ADR-0053](0053-at-least-once-outbox-poll.ja.md)。
