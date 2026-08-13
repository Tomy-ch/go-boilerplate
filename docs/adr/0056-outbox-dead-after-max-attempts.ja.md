---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [outbox, async]
---

# ADR-0056: MaxAttempts = 10 到達でメッセージをデッド状態にする（手動リプレイまで終端）

English canonical: [0056-outbox-dead-after-max-attempts.md](0056-outbox-dead-after-max-attempts.md)

## ステータス

accepted

## 背景

一部のアウトボックス行は、何回リトライしても成功しない — 例えば、恒久的に設定ミスのあるエンドポイント、レシーバーが無効として拒否するペイロード、または廃止されたレシーバーなど。リトライ上限がなければ、これらの行は永遠に `pending` のままで、ポールのたびにリレー容量を消費し、本物のデリバリー問題を隠すノイズでラグメトリクスを汚染する。

## 決定

`DefaultMaxAttempts = 10`。`MarkFailed` の呼び出しごとに `attempts` がインクリメントされ、`last_error` が設定される。`attempts >= MaxAttempts` になると、リレーは `MarkDead` を呼び出し、行を `dead` ステータスに遷移させる。デッド行は**終端**である。リレーはそれ以上のデリバリーを試みず、`outbox.dead` メトリクスカウンターがインクリメントされ、警告がログに出力される。回復は手動: オペレーターが `outbox-relay replay [--message-id=<uuid>]` を呼び出し、`attempts = 0` と `last_error = NULL` にリセットして行を `pending` に戻す。

許可されるステータス値は `pending`、`published`、`dead` の 3 つ（スキーマで CHECK 制約）。`failed` ステータスは存在しない。失敗した発行は、成功するか `MaxAttempts` を使い果たすまで `pending` のまま残る。

## 影響

### ポジティブな影響

- リレーループが、恒久的にデリバリー不能な行に容量を消費することから保護される。
- デッド行は `outbox.dead` メトリクスで露出され、観測可能でアラート設定が可能。
- 手動リプレイにより、正常な行を再処理することなく選択的な回復（単一メッセージまたは全バッチ）が可能。
- 各デッド行の `last_error` フィールドが診断のために最終的な失敗理由を記録する。

### ネガティブな影響

- 連続して 10 回以上一時的に失敗したメッセージは、オペレーターが手動でリプレイするまで隔離される。自動回復は行われない。
- `MaxAttempts = 10` は固定のデフォルト値。イベントタイプごとの上書きはない。

## 検討した代替案

### 無制限のリトライ

データ損失を防ぐが、デリバリー不能な行が無制限に蓄積し、ポール容量を消費して実際のラグを隠す。

### 使い果たした際に削除

最もシンプルな有界動作だが、データが永久に失われ、回復パスが存在しない。

### タイムアウト後のみデッドレターとする自動指数バックオフ

より洗練されたリトライエンベロープだが、行ごとのタイマー状態が必要で `FOR UPDATE SKIP LOCKED` のクレームモデルが複雑になる — バックオフ中の行はロックを消費せずにスキップしなければならない。シンプルな試行回数アプローチを優先して先送り。

## 補足

- `DefaultMaxAttempts`、`dead` 状態、手動リプレイは `docs/design/outbox.md`（§「State transitions」、用語集エントリ「MaxAttempts」と「dead」）に記述されている。
- 関連 ADR: [ADR-0053](0053-at-least-once-outbox-poll.ja.md)、[ADR-0057](0057-outbox-retention-gc.ja.md)。
