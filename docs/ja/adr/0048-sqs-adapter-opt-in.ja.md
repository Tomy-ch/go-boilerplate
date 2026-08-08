---
status: superseded
date: 2026-07-04
deciders: [maintainers]
superseded-by: 0048
tags: [worker, async, dependencies]
---

# ADR-0048: SQS アダプターはオプトインであり、デフォルトバイナリにリンクしない

English canonical: [0048-sqs-adapter-opt-in.md](../../adr/0048-sqs-adapter-opt-in.md)

## ステータス

superseded by [ADR-0049](0049-broker-sdk-isolation-measured-as-coupling.ja.md)

## 背景

ワーカースキャフォールド（[ADR-0046](0046-broker-agnostic-worker-scaffold.ja.md)）は AWS SQS 向けのリファレンスブローカーアダプターを同梱している。このアダプターをデフォルトビルドに組み込むと、キューを消費しない `serve` を含むすべてのバイナリが `aws-sdk-go-v2` をリンクし、ロックイン回避原則（[ADR-0001](0001-avoid-lock-in.ja.md)）に反して依存性サーフェスが拡大する。

## 決定

SQS アダプターを**オプトイン**のままにする。リファレンスアダプターは `internal/infrastructure/queue/sqs` に置き、**デフォルトの `cmd` ビルドには組み込まない**。これにより `aws-sdk-go-v2` は出荷バイナリにリンクされない。SQS を使いたいデプロイメントは明示的にアダプターを組み込む。

## 影響

### ポジティブな影響

- デフォルトバイナリは AWS SDK から解放される — 依存性サーフェスとビルドが軽量になる。
- リファレンスアダプターは任意のプル・アック型ブローカーの実装例として引き続き存在する。
- ビルドタグなしで、`cmd` からインポートしないことで依存性の分離が達成される。

### ネガティブな影響

- SQS を有効化するには明示的な組み込み手順が必要。デフォルトでは有効にならない。

## 検討した代替案

### デフォルトで SQS を組み込む

却下: `aws-sdk-go-v2` を（`serve` を含む）すべてのバイナリに強制し、交換可能なインフラという目標に反する。

### 依存性分離のためのビルドタグ

却下: このリポジトリに前例がなく、シングルバイナリではモジュール分離が不十分。`cmd` からアダプターをインポートしないことで、タグなしで依存性が分離できる。

## 補足

- 親決定: [ADR-0046](0046-broker-agnostic-worker-scaffold.ja.md)。原則: [ADR-0001](0001-avoid-lock-in.ja.md)。
- 参考: `internal/infrastructure/queue/sqs/README.md`。
- `docs/decisions.md`（§「Why a broker-agnostic worker scaffold」— dependency isolation）から移行。
