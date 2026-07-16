---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [outbox, async, gc]
---

# ADR-0049: 発行済み行の 7 日間保持 GC（10,000 件単位のバッチ）

English canonical: [0049-outbox-retention-gc.md](../../adr/0049-outbox-retention-gc.md)

## ステータス

accepted

## 背景

アウトボックス行が `published` とマークされた後は、それ以上のデリバリー目的を果たさない。クリーンアップポリシーがなければ `outbox` テーブルが際限なく成長し、ストレージコストが増加し、テーブルサイズの増大に伴って `ClaimPending` クエリが徐々に遅くなる — `status = 'pending'` の部分インデックスがあっても、テーブルの肥大化は VACUUM の効率やページキャッシュ効率を低下させる。

## 決定

`GCUsecase.SweepPublished` は `published_at` タイムスタンプが `DefaultRetention = 7 日` より古い `published` 行を削除し、1 回の呼び出しで最大 `DefaultGCBatchSize = 10,000` 行を処理する。基底の SQL は `published_at` 順に候補を選択し `id IN (サブクエリ)` で削除することで、ステートメントごとのロック時間を制限する。GC は `cmd job outbox-gc` — 外部 cron でスケジュールされるワンショットコマンド — 経由で呼び出される（[ADR-0051](0051-relay-resident-gc-oneshot.ja.md) を参照）。

SQL は以下の通り:

```sql
DELETE FROM outbox
WHERE id IN (
    SELECT o.id
    FROM outbox AS o
    WHERE o.status = 'published'
      AND o.published_at < $1
    ORDER BY o.published_at
    LIMIT $2
);
```

## 影響

### ポジティブな影響

- `outbox` テーブルが有界になる。単調に増加しない。
- バッチ削除によりステートメントごとのロック時間と I/O プレッシャーが制限され、並行するリレー処理への影響が軽減される。
- 7 日間の保持期間により、データを際限なく蓄積することなく、デバッグのために直近のデリバリー履歴が保持される。

### ネガティブな影響

- 発行済み行はデリバリー後最大 7 日間スペースを占有する。これは許容できるが、テーブルに常に一部の過去の行が含まれることを意味する。
- バッチサイズと保持期間は固定のデフォルト値。変更するにはコード変更が必要。
- 外部スケジューラー（cron または Kubernetes CronJob）をプロビジョニングして監視しなければならない。設定ミスや不在の cron では発行済み行が蓄積される。

## 検討した代替案

### MarkPublished 内での削除

即時クリーンアップだが、すべてのマーク呼び出しにレイテンシを追加し、リレートランザクション内で実行されるため行ロック保持時間が長くなる。

### リレーに組み込んだ継続的バックグラウンドスイーパー

外部スケジューラーへの依存を排除できるが、関係のない 2 つの懸念を 1 つのプロセスに結合し、すでにステートフルなループに並行性を追加する。

### GC なし（永続保持）

運用上はシンプルだが、非自明なイベント量では持続不能。

## 補足

- `DefaultRetention`（7 日）と `DefaultGCBatchSize`（10,000）は `docs/design/outbox.md`（用語集エントリ「GC (SweepPublished)」）に記述されている。
- SQL ソース: `database/dml/system_cqrs/outbox/delete_published_outbox.sql`。
- 関連 ADR: [ADR-0048](0048-outbox-dead-after-max-attempts.ja.md)、[ADR-0051](0051-relay-resident-gc-oneshot.ja.md)。
