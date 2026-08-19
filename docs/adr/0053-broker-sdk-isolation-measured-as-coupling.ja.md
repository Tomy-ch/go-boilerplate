---
status: accepted
date: 2026-08-08
deciders: [maintainers]
supersedes: 0047
tags: [worker, outbox, async, dependencies]
---

# ADR-0053: ブローカー SDK の分離はリンクではなく結合で測る

English canonical: [0053-broker-sdk-isolation-measured-as-coupling.md](0053-broker-sdk-isolation-measured-as-coupling.md)

## ステータス

accepted（[ADR-0052](0052-sqs-adapter-opt-in.ja.md) を supersede）

## 背景

[ADR-0052](0052-sqs-adapter-opt-in.ja.md) は、出荷バイナリに `aws-sdk-go-v2` をリンクさせないために、SQS リファレンスアダプターを `cmd` のデフォルト配線から外していた。[`docs/design/worker.md`](../design/worker.ja.md) はその結果生まれた不変条件を **E3**（「出荷バイナリはブローカー SDK を含まない」）として記録している。この記録が書かれて以降、2 つのことが変わった。

**前提がドリフトした。** 中立な `ObjectStorage` 境界とその S3 アダプターが ADR-0052 の accepted 後に追加された。`github.com/aws/aws-sdk-go-v2/service/s3` と、それに伴う SDK コア（`aws` / `credentials` / `signer/v4` / `retry` / `transport/http`）および `smithy-go` は、現在デフォルトバイナリにリンクされている。ADR-0052 の背景（「キューを消費しない `serve` を含むすべてのバイナリが `aws-sdk-go-v2` をリンクする」）は執筆時点では正確だったが、現在は成り立たない。今日 `internal/infrastructure/queue/sqs` を配線して増えるのは、`service/sqs` / `service/sqs/internal/endpoints` / `service/sqs/types` / `aws/protocol/restjson` / `smithy-go/encoding/json` の 5 パッケージだけである。

**機構がバイナリ構成と噛み合っていない。** 本リポジトリは Cobra のサブコマンド（`serve` / `worker` / `outbox-relay` …）を持つ単一バイナリを出荷する。リンクはバイナリ単位、役割はサブコマンド単位なので、`worker` が同じバイナリにいる限り「`serve` をキュー SDK から遠ざける」は import の統制では達成できない。したがって E3 が実際に保証しているのは軽量な `serve` ではなく、**同梱のリファレンスアダプターが決して配線されず、ゆえに一度も端から端まで実行されない**という状態である。エンジン・シーム・アダプターはいずれもビルドされ単体テストもされているが、本リポジトリでメッセージが通過したことは一度もなく、`internal/di/module/worker.go` の `provideWorkers()` は空のままである。アダプター固有の挙動（可視性タイムアウトの延長、receipt handle の往復、属性マッピング、`ApproximateReceiveCount` のパース）は、モック化したクライアントに対してしか検証されていない。

**同じ問いは送出側にも生じる。** `publisher.Publisher` の実装は HTTP POST の 1 つだけで、core のファイルがブローカーアダプターを参照することは無かった。outbox の publish 先にキューを与えるということは、core のファイル — `internal/infrastructure/publisher` にある実装セレクター — がそれを import するということである。流れる向きは違っても問いの形は同じなので、以下は worker だけでなく両方のシームに対して述べる。

ADR-0052 の背後にある懸念は、リンクではなく**結合**として述べたほうが正確である。すなわち、リポジトリが単一ベンダーに構造的に依存してはならない、ということだ。リンクはその代理指標として粗い。差し替え可能性を保っているのはシーム（`worker.Consumer` / `publisher.Publisher`）であり、アダプターを import しないことはそこに何も足さない。取り除いているのは動く実例だけである。

## 決定

ブローカーアダプターは、キューのどちら側であっても、**デフォルトビルドへ配線してよい**。受信側のシーム（`worker.Consumer` / `worker.FailureHandler`）と outbox の publish 側のシーム（`publisher.Publisher`）は同じ扱いとする。E3 は、リンクではなく結合を測る形へ述べ直す。

> **E3**: 具体的なブローカーの知識は、そのアダプターのパッケージと、それを選ぶ配線だけに閉じ込める。core の `*.go` も core のドキュメントもブローカーアダプターを名指さず、core のコードが名指すのはシームだけである。

E3 は機械的に検証できる。リンカが何を出力したかではなく、どのファイルが何を名指しているかについての言明だからである。`*.go` 側の検査は `scripts/setup/verify-sample-removal.ts` の `checkNoDanglingReferences` が担い、core のドキュメントは構造（`internal/controller/worker/<name>/`）を記述して具体アダプターを名指さない。

ベンダーを持ち込むのは**配線**であってアダプターではない。アダプターのパッケージ、本物の代わりに立てるローカルブローカーのサービス、およびそれらが読む設定は、自分のベンダーだけを名指し、他のどこからも到達されない — object storage のアダプターとローカルの Garage サービスが既にそうであるように。core のファイルをブローカーアダプター参照の立場に置くものは、ちょうど 3 つ、import・判別子の分岐・それを選ぶ値である。ベンダーをそこへ閉じ込めることが差し替えを有界な変更にし、使っていないアダプターを削除せずビルドと単体テストまで行ったうえで残す理由でもある。

**E1 / E2 は変更しない**。エンジンは infrastructure を import せず、インメモリ fake だけでグリーンになる。

## 影響

### ポジティブな影響

- サブシステムが端から端まで実行された経路を得る。アダプターと配線テンプレートが「一度も動いたことのないコード」でなくなる。
- E3 は自動で強制できるし、実際に強制されている。リンクの形では強制できなかった。`depguard` にも `internal/architest/**` にも該当ルールが無く、どちらでも表現できなかったためである。
- ブローカーの差し替えが有界な変更になる。import・判別子の分岐・それを選ぶ値だけで済み、他所に探すものがない。

### ネガティブな影響

- アダプターを配線するとその SDK がリンクされるため、別のブローカーを狙うデプロイは、配線を変えるまで使わないパッケージを抱える。
- 配線を戻せるようにするなら、シームの配線前の形をどこかに取り出せる状態で保つ必要があり、単一の形より読みにくい。

### 中立的な影響

- アダプターがビルド・テストされている以上、`go.mod` は既に `service/sqs` を直接依存として宣言している。本決定が変えるのは *リンクされるもの* であって *必要とされるもの* ではない。

## 検討した代替案

### E3 をリンクの形のまま維持する

配線された実例を一切禁じることになり、worker サブシステムはエンジンとシーム、そして誰も使ってはならないアダプターに縮む。却下: 実演された経路はわずかに小さいバイナリより価値がある。またリンクの規則が買う分離は、その根拠となる原則が本来求めている分離ではない。

### バイナリを分割する

`serve` と `worker` を別の `main` パッケージにすればリンクが役割単位になり、リンクの形を文字どおり維持できる。現時点では却下: ADR-0052 は同じ目的でビルドタグを既に却下しており、サブコマンドを持つ単一イメージというデプロイモデルに対して、バイナリ分割は目的に対して大きすぎる構造変更である。出荷上の制約が要求する場面には引き続き選択肢として残る。

### SDK を必要としないブローカーで実演する

Postgres ベースの pull-ack アダプターであれば、ベンダー SDK をリンクせずにシームを動かせるうえ、シームが SQS 専用の形ではないことも証明できる。却下: 実例のために 2 本目のアダプターを作り保守することになる一方、E3 が既に配線された実例の持ち込む結合を上限で抑えている。ローカルブローカーが SQS 互換であるため、リファレンスアダプターも再利用できる。

## 補足

- [ADR-0052](0052-sqs-adapter-opt-in.ja.md) を supersede する。親の決定: [ADR-0050](0050-broker-agnostic-worker-scaffold.ja.md)。原則: [ADR-0001](0001-avoid-lock-in.ja.md)。
- [ADR-0051](0051-out-of-scope-push-streaming-brokers.ja.md) は影響を受けない。push 型・ストリーミングログ型のブローカーは引き続き worker ポートの対象外である。outbox の publish 先として pull-ack ブローカーを選ぶことは、そもそも同 ADR の対象ではなかった。
- E3 は [`docs/design/worker.md`](../design/worker.ja.md) に記載する。
- 参照: アダプターは [`internal/infrastructure/queue/sqs/README.md`](../../internal/infrastructure/queue/sqs/README.ja.md)、それを選ぶ判別子は [`internal/infrastructure/publisher/README.md`](../../internal/infrastructure/publisher/README.ja.md)。
