---
status: accepted
date: 2026-08-04
deciders: [maintainers]
supersedes: 0044
tags: [worker, async, dependencies]
---

# ADR-0106: ブローカー SDK の分離は、アダプターを未配線にすることではなくサンプル削除後に検証する

English canonical: [0106-broker-sdk-isolation-verified-after-sample-removal.md](../../adr/0106-broker-sdk-isolation-verified-after-sample-removal.md)

## ステータス

accepted（[ADR-0044](0044-sqs-adapter-opt-in.ja.md) を supersede）

## 背景

[ADR-0044](0044-sqs-adapter-opt-in.ja.md) は、出荷バイナリに `aws-sdk-go-v2` をリンクさせないために、SQS リファレンスアダプターを `cmd` のデフォルト配線から外していた。[`docs/design/worker.md`](../design/worker.ja.md) はその結果生まれた不変条件を **E3**（「出荷バイナリはブローカー SDK を含まない」）として記録している。この記録が書かれて以降、2 つのことが変わった。

**前提がドリフトした。** 中立な `ObjectStorage` 境界とその S3 アダプターが ADR-0044 の accepted 後に追加された。`github.com/aws/aws-sdk-go-v2/service/s3` と、それに伴う SDK コア（`aws` / `credentials` / `signer/v4` / `retry` / `transport/http`）および `smithy-go` は、現在デフォルトバイナリにリンクされている。ADR-0044 の背景（「キューを消費しない `serve` を含むすべてのバイナリが `aws-sdk-go-v2` をリンクする」）は執筆時点では正確だったが、現在は成り立たない。今日 `internal/infrastructure/queue/sqs` を配線して増えるのは、`service/sqs` / `service/sqs/internal/endpoints` / `service/sqs/types` / `aws/protocol/restjson` / `smithy-go/encoding/json` の 5 パッケージだけである。

**機構がバイナリ構成と噛み合っていない。** 本リポジトリは Cobra のサブコマンド（`serve` / `worker` / `outbox-relay` …）を持つ単一バイナリを出荷する。リンクはバイナリ単位、役割はサブコマンド単位なので、`worker` が同じバイナリにいる限り「`serve` をキュー SDK から遠ざける」は import の統制では達成できない。したがって E3 が実際に保証しているのは軽量な `serve` ではなく、**同梱のリファレンスアダプターが決して配線されず、ゆえに一度も端から端まで実行されない**という状態である。エンジン・シーム・アダプターはいずれもビルドされ単体テストもされているが、本リポジトリでメッセージが通過したことは一度もなく、`internal/di/module/worker.go` の `provideWorkers()` は空のままである。アダプター固有の挙動（可視性タイムアウトの延長、receipt handle の往復、属性マッピング、`ApproximateReceiveCount` のパース）は、モック化したクライアントに対してしか検証されていない。

ADR-0044 の背後にある懸念は、リンクではなく**結合**として述べたほうが正確である。すなわち、リポジトリが単一ベンダーに構造的に依存してはならない、ということだ。リンクはその代理指標として粗い。差し替え可能性を保っているのはシーム（`worker.Consumer` / `publisher.Publisher`）であり、アダプターを import しないことはそこに何も足さない。取り除いているのは動く実例だけである。

## 決定

ブローカーアダプターは、**削除可能なサンプル群の一部としてデフォルトビルドへ配線してよい**。E3 は、リンクではなく結合を測る不変条件へ置き換える。

> **E3'**: `make setup-remove-sample-api` の実行後、リポジトリの結合はサンプル追加前と同一である。

E3' は 4 つの条件に分解でき、いずれも機械的に検証できる。

1. **core の `*.go` がブローカーアダプターを参照しない** — `scripts/setup/verify-sample-removal.mjs` の `checkNoDanglingReferences` が検査する。
2. **core のドキュメントがサンプルを参照しない** — core のドキュメントは構造（`internal/controller/worker/<name>/`）を記述し、サンプルの具体名を参照しない。
3. **シームがサンプル追加前の形へ戻る** — `internal/usecase/boundary/worker` へのサンプル由来の変更は、退避側にサンプル追加前の形を保持する `sample-api:replace` ブロックで囲む。
4. **不要になった依存が `go.mod` / `vendor/` から落ちる** — 削除チェーンで `make tidy-lib` を実行する。

**E1 / E2 は変更しない**。エンジンは infrastructure を import せず、インメモリ fake だけでグリーンになる。

## 影響

### ポジティブな影響

- サブシステムが端から端まで実行された経路を得る。アダプターと配線テンプレートが「一度も動いたことのないコード」でなくなる。
- E3' はプルリクエストごとに強制される。`.github/workflows/sample-removal-check.yaml` は既にフル削除に続けて `go build ./...` / `make lint` / `make test` を実行しており、削除後の状態が継続的に検証される。E3 には自動強制が一切なかった（`depguard` にも `internal/architest/**` にも該当ルールが無い）。
- 条件 3 の退避側は腐らない。同じワークフローが毎回コンパイルしテストするためである。
- 条件 2 により、core の設計ドキュメントが他サブシステムのサンプルを参照しているという既存の不整合が解消される。

### ネガティブな影響

- AWS 以外のブローカーを使う fork は、サンプル削除を実行するまで SQS の 5 パッケージを引き継ぐ。
- サンプル由来のシーム変更は 1 つのファイル内に 2 つの形（有効側と退避側）で存在し、単一の形より読みにくい。

### 中立的な影響

- アダプターがビルド・テストされている以上、`go.mod` は既に `service/sqs` を直接依存として宣言している。本決定が変えるのは *リンクされるもの* であって *必要とされるもの* ではない。

## 検討した代替案

### E3 を現状のまま維持する

配線された実例を一切禁じることになり、worker サブシステムはエンジンとシーム、そして誰も使ってはならないアダプターに縮む。却下: テンプレートにとって、実演された経路はわずかに小さいバイナリより価値がある。また E3 が買う分離は、その根拠となる原則が本来求めている分離ではない。

### バイナリを分割する

`serve` と `worker` を別の `main` パッケージにすればリンクが役割単位になり、E3 を文字どおり維持できる。現時点では却下: ADR-0044 は同じ目的でビルドタグを既に却下しており、サブコマンドを持つ単一イメージというデプロイモデルを採るテンプレートに対して、バイナリ分割は目的に対して大きすぎる構造変更である。出荷上の制約が要求する fork には引き続き選択肢として残る。

### SDK を必要としないブローカーで実演する

Postgres ベースの pull-ack アダプターであれば、ベンダー SDK をリンクせずにシームを動かせるうえ、シームが SQS 専用の形ではないことも証明できる。却下: サンプルのために 2 本目のアダプターを作り保守することになる一方、E3' が既にサンプルの持ち込む結合を上限で抑えている。ローカルブローカーが SQS 互換であるため、リファレンスアダプターも再利用できる。

## 補足

- [ADR-0044](0044-sqs-adapter-opt-in.ja.md) を supersede する。親の決定: [ADR-0042](0042-broker-agnostic-worker-scaffold.ja.md)。原則: [ADR-0001](0001-avoid-lock-in.ja.md)。
- [ADR-0043](0043-out-of-scope-push-streaming-brokers.ja.md) は影響を受けない。push 型・ストリーミングログ型のブローカーは引き続き worker ポートの対象外である。outbox の publish 先として pull-ack ブローカーを選ぶことは、そもそも同 ADR の対象ではなかった。
- E3' は [`docs/design/worker.md`](../design/worker.ja.md) に記載し、`.github/workflows/sample-removal-check.yaml` が強制する。
- 参照: [`internal/infrastructure/queue/sqs/README.md`](../../../internal/infrastructure/queue/sqs/README.ja.md)。
