---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [outbox, http]
---

# ADR-0049: パブリッシャーの非標準 HTTP プロファイルをリレー内に隔離する

English canonical: [0049-publisher-http-profile-isolation.md](../../adr/0049-publisher-http-profile-isolation.md)

## ステータス

accepted

## 背景

アウトボックス HTTP パブリッシャーは、共有デフォルト HTTP クライアントプロファイルから次の 3 点で逸脱する必要がある。

1. **`MaxAttempts = 1`** — トランスポートレベルのリトライは無効にしなければならない。リレーのポールループが at-least-once リトライ機構であるため（[ADR-0044](0044-at-least-once-outbox-poll.ja.md) 参照）、両方を有効にするとリトライが二重増幅してしまう（決定 D10）。

2. **`PropagateTrace = false`** — W3C の `traceparent` ヘッダーは emit 時点でキャプチャされ、`headers` カラムにそのまま保存される。パブリッシャーはその保存値をリプレイすることで、受信側が元のトレーススパンに接続できるようにする。送信時に自動でトレースコンテキストを注入すると、保存済みの `traceparent` がリレー自身のスパンで上書きされ、トレースの連続性が断ち切られる。

3. **`AllowPrivateNetwork = false`** — 受信エンドポイントは外部サービスである。プライベートアドレスやループバックアドレスへの配信を拒否することで SSRF リスクを抑制する（耐障害性と SSRF クライアントポリシーについては ADR-0019、ADR-0020 を参照）。

このプロファイルを共有の `InfrastructureModule` に登録すると、そのモジュールを使用するすべてのプロセス（メイン API サーバーを含む）の DI グラフに組み込まれ、他のダウンストリームクライアントが使用する標準 HTTP プロファイルを上書き・競合させる可能性がある。

## 決定

`outboxPublisherModule`（`NewDownstreamProfile` を通じて非標準の `DownstreamProfile` を登録する）は **`OutboxRelayModule` 内にネスト**され、リレー専用プロセス（`cmd outbox-relay`）のみで組み立てられる。メインサーバーの `InfrastructureModule` には含まれない。ネスト構造により、非標準プロファイルが他のプロセスに漏洩することはない。

```text
OutboxRelayModule
└── outboxPublisherModule          ← 非標準 DownstreamProfile を登録
    └── provideHTTPClientProfiles  ← value group に追加
```

## 影響

### ポジティブな影響

- 非標準プロファイルはリレープロセスに厳密にスコープされ、他のダウンストリーム HTTP クライアントに誤って影響を与えることがない。
- プロファイルの所有権がリレーエントリポイントと同じ場所に置かれ、`http_publisher.go` を読まずとも DI グラフだけで制約を監査できる。
- 共有 `httpclient` インフラのオブザーバビリティ（メトリクス、トレーシング）はプロファイルを損なわずパブリッシャーから利用可能なまま維持される。

### ネガティブな影響

- リレープロセスはメインサーバーグラフのスーパーセットである独自の DI グラフを持つため、合成コードが大きくなる。
- パブリッシャー関連モジュールを追加する開発者は、正しい `fx.Module` に配置するためにリレー/サーバーの分割を把握しておく必要がある。

## 検討した代替案

### フィーチャーフラグを使って InfrastructureModule にプロファイルを登録する

ランタイム設定のサーフェスが増え、プロセス ID ではなく環境変数によってプロファイルが条件付きになる。これはより弱い保証である。メインサーバーでフラグを誤設定すると誤ったプロファイルが有効化される可能性がある。

### 共有プロファイルシステムで管理しない独立した HTTP クライアントインスタンスを使う

共有オブザーバビリティ（RED メトリクス、トレーシング）が失われ、プロファイルシステムが提供するポリシー強制（SSRF、サーキットブレーカー、バジェット）も回避される。

## 補足

- 非標準プロファイルの根拠: `docs/design/outbox.md`（§「パッケージ配置と依存関係の方向」、用語集エントリ「OutboxRelayModule」）。
- 実装:
  - `internal/di/module/outboxpublisher.go`（`outboxPublisherModule`、`NewDownstreamProfile`）
  - `internal/di/module/outboxrelay.go`（`OutboxRelayModule`）
  - `internal/infrastructure/publisher/http_publisher.go`（`NewDownstreamProfile`）
- 耐障害性と SSRF クライアントポリシー: ADR-0019、ADR-0020（プレーンテキスト — 未公開）。
- 関連 ADR: [ADR-0044](0044-at-least-once-outbox-poll.ja.md)、[ADR-0050](0050-relay-resident-gc-oneshot.ja.md)。
