---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [outbox, async, idempotency]
---

# ADR-0044: アウトボックスの message_id をレシーバーの Idempotency-Key として伝播する

English canonical: [0044-message-id-idempotency-propagation.md](../../adr/0044-message-id-idempotency-propagation.md)

## ステータス

accepted

## 背景

リレーは少なくとも1回のセマンティクスでメッセージをデリバリーする（[ADR-0042](0042-at-least-once-outbox-poll.ja.md) を参照）。そのため、一時的なデリバリー失敗、プロセス再起動、またはエッジケースの重複クレームにより、レシーバーが同じメッセージを複数の HTTP 呼び出しで受け取ることがある。アプリケーションレベルの調整なしにレシーバーが重複排除できるようにするため、すべてのデリバリーは、トランスポート試行ではなく論理イベントを識別する、安定したグローバルにユニークなキーを持つ必要がある。

## 決定

各アウトボックス行には `message_id` UUID が **INSERT 時に一度だけ**割り当てられる（`EmitUsecase.Emit` が割り当てる）。HTTP パブリッシャーはレシーバーエンドポイントへの POST のたびに、この UUID を `Idempotency-Key` リクエストヘッダーとして伝播する。レシーバーはこのキーを重複排除トークンとして扱い、イベントが永続的に受け入れられた後にのみ `2xx` を返すことが期待される。

## 影響

### ポジティブな影響

- キーはリトライをまたいで安定している。何回のポールサイクルが経過したかに関わらず、特定のアウトボックス行のすべてのデリバリー試行で同じ UUID が送信される。
- レシーバー側の重複排除がシンプルになる。`Idempotency-Key` にキーを張ったインデックス付きカラム 1 つで、少なくとも1回のデリバリーを正確に1回の効果に変換できる。
- キーはエミット時に同期的に割り当てられるため、行がデリバリーされる前からトレーシングとデバッグに利用できる。

### ネガティブな影響

- レシーバーは重複排除を実装しなければならない。このサブシステムはキーを提供するが、レシーバー側のストレージやロジックは提供しない。
- `message_id` は INSERT 後に変更されてはならない。UUID を再生成するような変更（例: 新しい UUID を割り当てるリプレイ）はレシーバーの重複排除を破壊する。

## 検討した代替案

### ペイロードハッシュによるレシーバー側重複排除

ペイロードにタイムスタンプやその他の非決定論的フィールドが含まれる場合は脆弱。スキーマバージョン間のペイロード進化を生き残れない。

### 重複排除メカニズムなし

サブシステム提供の安定したリファレンスキーなしに、すべての負担をレシーバーに押し付ける。少なくとも1回のデリバリーでは非現実的。

## 補足

- レシーバーの冪等性設計不変条件: `docs/design/outbox.md`（§「Design invariants」、インテグレーターチェックリスト ③）。
- `message_id` の伝播は `internal/infrastructure/publisher/http_publisher.go` の `Publish`（`httpclient.WithIdempotencyKey`）で実装されている。
- 関連 ADR: [ADR-0041](0041-transactional-outbox.ja.md)、[ADR-0042](0042-at-least-once-outbox-poll.ja.md)。
