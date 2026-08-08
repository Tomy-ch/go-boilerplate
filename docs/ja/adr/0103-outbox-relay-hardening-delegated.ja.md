---
status: accepted
date: 2026-07-10
deciders: [maintainers]
tags: [exclusion, outbox, messaging, reliability, setup-review]
---

# ADR-0103: outbox relay の重複窓ハードニングは本番コピー側の責務とする

English canonical: [0103-outbox-relay-hardening-delegated.md](../../adr/0103-outbox-relay-hardening-delegated.md)

## ステータス

accepted

## コンテキスト

outbox relay は pending 行を claim し、HTTP で publish し、その結果を**単一トランザクション内**で
mark する（[ADR-0051]、[ADR-0052]）。この種の設計はすべて不可能性の結果に縛られる: 外部副作用
（HTTP POST）とその記録（DB 行）は原子化できない（Two Generals）。重複・喪失・可用性の
*窓（window）* — 不変条件が一時的に破られうる、始点と終点を持つ時間区間 — を同時にゼロには
できず、設計とは「どの窓を残すか」の選択である。窓への対処は 3 種類しかない: **閉じる**
（交錯を構造的に不可能にする）、**狭める**（区間を縮める）、**吸収する**（下流で無害化する）。

出荷状態の単一トランザクション relay は以下の窓を持つ:

- **tx 滞留**: publish が claim tx の内側で走るため、最悪 tx 時間は
  `BatchSize (100) × 試行あたりタイムアウト (~3s) ≈ 300s`。vacuum horizon をピン留めし、pool
  接続を 1 本占有する。この帰結として、pool 全体への `idle_in_transaction_session_timeout` の
  バックストップは意図的に見送っている（`driver.applyDBTimeouts` は `statement_timeout` /
  `lock_timeout` のみを設定する）: 暴走トランザクションをバックストップできるほど短い一律値は、
  relay 自身の長命な claim→publish→mark tx を kill してしまう。pool 全体への有効化が安全に
  なるのは、設計図の第 2 層が publish をトランザクション外へ移した後である。
- **per-message backoff の不在**: max attempts はハードコードされ（`DefaultMaxAttempts = 10`）、
  失敗行は次 poll（既定 1s）で再 claim される。数十秒の下流停止だけで pending 全量が `dead` へ
  落ちる。
- **tx リトライ再送**: serialization failure / deadlock リトライ（[ADR-0031]）は claim →
  publish → mark の関数全体を再実行し、配送済みメッセージを同一 poll 内で再送する。relay は
  ドレイン本体であり退避先の outbox を持たないため、「外部副作用は outbox 行に置く」という
  ADR-0031 の規則に対する唯一の公認例外である。
- **attempts の下限保証化**: バッチ rollback は `attempts` の加算を消すが、実行済みの HTTP POST
  は消えない。
- **lag SLI の曖昧さ**: publish 中の行も `pending` のままカウントされ、lag gauge は処理中と滞留を
  区別できない。

閉じられる窓をすべて閉じるハードニング設計は存在する（下記設計図）。テンプレートに実装する圧力は
現実にある。しかしそれはスキーマ（列 2 本 + status 値）、実行トポロジへのリーダー選出依存、そして
デプロイ固有のチューニング値（lease 長・deadline margin・backoff 曲線）を持ち込む — テンプレートが
知り得ないポリシーであり、[ADR-0100] と [ADR-0101] と同じ論法が当てはまる。

## 決定

このテンプレートでは outbox relay のハードニングを**意図的に実装しない**。テンプレートは単純な
単一トランザクション relay と、同梱の吸収契約 — `message_id` を `Idempotency-Key` として伝搬
（[ADR-0053]）し、受信側の同梱冪等性ミドルウェアが dedup する — を維持する。受信側も本テンプレート
から作られていれば重複は exactly-once *effect* に構造的に畳まれる（第三者実装の受信者は dedup 義務を
統合要件として引き継ぐ）。

outbox を本番スケールで運用するコピーは、以下の多層構造で relay を再設計する **SHOULD**。この
構造に機能的トレードオフはなく、対価は実装コストと可用性窓である。

### ハードニング設計図

1. **lease 方式の claim**: `claiming` status・`claimed_until`・`next_attempt_at` を追加し、失効
   lease の回収を claim 述語（`pending AND next_attempt_at <= now()` OR
   `claiming AND claimed_until < now()`）に畳み込む — 専用 reaper プロセスを持たない。
2. **publish の tx 外化**: claim と mark は短 tx になる。tx 滞留・vacuum ピン留め・
   idle-in-transaction の懸念・tx リトライ再送経路が消える。どのトランザクションにも外部副作用が
   含まれなくなるため、relay は ADR-0031 の一般則を満たし、公認例外そのものが消滅する。
3. **per-message 指数 backoff**（`next_attempt_at`）+ max attempts の設定化 — 下流停止時の
   dead 化は数十秒から分〜時間オーダーへ。
4. **singleton トポロジ**: セッションスコープの Postgres advisory lock で、インスタンス間の併走
   publish をトポロジレベルで閉じる。保持セッションの死でロックは自動解放される。
5. **自己 deadline fence**: 各バッチの publish を `claimed_until − margin` で打ち切る。基準時刻は
   claim 時に取得した DB 時計に固定する — 生存インスタンスは自分の lease の外で構造的に行動
   できない（アプリ間 clock skew に依存しない）。
6. **fence 付き mark**: `WHERE status = 'claiming' AND claimed_until = <自 lease>` — 二重 mark は
   DB 内で完全に閉じる。
7. **最終層としての吸収**: 残る重複は crash-class・時間差（併走しない）・回数有界・同一
   `message_id` キー — 同梱の冪等性ミドルウェアが決定論的に畳める形そのものである。

この構造の下では、正常運転中の重複は構造的にゼロになる。lease 長はクラッシュ回収の遅延だけを
決める運用値になり（重複確率とのトレードオフではなくなる）、残る対価は可用性窓（failover の
引き継ぎ、ハング保持者の検知）で、これはデプロイ側 runbook の領分である。

### バランス型の委譲

この委譲は、ハードニングのコストをテンプレートとコピーの間で分担する:

- **判定文ではなく設計図**: 本 ADR はスキーマ・claim 述語・fence・残余窓の台帳という機械的設計の
  全体を保持する。コピーが払うのは実装とチューニングのコストであり、分析のやり直しではない。
- **書き直しではなく拡張継ぎ目**: ハードニング設計はテンプレートの既存の継ぎ目に収まるため、
  コピーはフォークして書き直すのではなく、特定のインターフェースとテーブルの拡張として構築する:
  - *戦略の継ぎ目*: relay engine は poll ループと待機だけを持ち、claim → publish → mark の業務
    には `RelayUsecase` インターフェース経由でのみ到達する。ハードニング版の編成はこの
    インターフェースの新実装であり、DI で選択される。
  - *永続化の継ぎ目*: `boundary/outbox.Store` は lease 設計と**シグネチャ互換** —
    `ClaimPending(ctx, limit)` は lease claim の `UPDATE … RETURNING` で実装可能で、mark 系
    メソッドも形を保つ。契約は挙動（並行呼び出しが同一行を二重取得しない）で記述され、機構
    （`FOR UPDATE` という文言）では記述されないため、lease 実装は契約から逸脱するのではなく
    **適合**する。
  - *クエリの継ぎ目*: 1 クエリ 1 SQL ファイル — 新しい述語は `database/dml/system_cqrs/outbox/`
    配下の新規ファイルとして追加される。
  - *スキーマの継ぎ目*: 新列（`next_attempt_at`、`claimed_until`）は追加 migration として届く。
- **非加算な箇所の正直な列挙**: `claiming` を許可するための `outbox_status_check` CHECK の
  差し替えと、新しい claim 述語向けの `outbox_pending_idx` の張り替え（いずれも新規 migration
  ファイル経由 — スキーマ進化の標準機構。CHECK は「拡張」できない）、および DI の provide 1 行の
  swap。テンプレートは CHECK に `claiming` を先行許可することも、使われない戦略スイッチを同梱
  することも意図的にしない — スキーマと配線は自身のコードが実際に行使するものだけを宣言する。
- **格上げトリガー**: ハードニング済みコピーが例外ではなく主流になった場合は、本設計図を
  `docs/design/` の実装ガイド（または参照実装）へ格上げし、本 ADR を再訪する — この除外は
  スキャフォールドの想定利用者に向けた既定であり、上限ではない。

## 帰結

### ポジティブな帰結

- テンプレートの relay は小さく読みやすいまま保たれ、exactly-once を装わず at-least-once 契約を
  正直に教える。
- コピーは、テンプレートが選んだ中途半端な既定値ではなく、完全に分析済みで機械的に適用可能な
  設計図（スキーマ・述語・fence・残余窓の台帳）を得る。
- 責務が明示される: テンプレートは吸収契約と本分析を持ち、コピーはハードニングとその運用
  チューニングを持つ。

### ネガティブな帰結

- テンプレートの relay には列挙した窓が残る: 多インスタンス運用では正常運転中にも重複が起こりえ、
  短い下流停止で backlog が dead 化する。出荷既定は低ボリュームまたは単一インスタンスの relay に
  適する。
- コピーが本番化するには実装作業（migration、claim/mark 書き換え、lock、fence、テスト）が必要で、
  実施後はテンプレートとの乖離が広がる — 上記の設計図と拡張継ぎ目により乖離は列挙可能な面に
  限定されるが、消えはしない。
- 第三者実装の受信者を守るのは文書化された dedup 義務のみで、本リポジトリからは強制できない。

## 検討した代替案

### 多層再設計をテンプレートに実装する

閉じられる窓をすべて出荷時に閉じられるが、デプロイ固有のポリシー（lease・margin・backoff・
singleton トポロジ）を、正しい値を知り得ないスキャフォールドに焼き込み、コアサンプルの学習面積を
広げる。テンプレートでは不採用とし、コピー側の作業が機械的になるよう本 ADR に記録する。

### 狭めるだけのチューニング（バッチ縮小・タイムアウト短縮・lease 値調整）

すべての窓が開いたまま幅と頻度を下げるだけ — 構造的ではなく確率的な防御。主戦略としては不採用。
狭める手段は閉じる手段の補完としてのみ有用である。

### at-most-once relay（送信前に published を記録し、結果不明の送信をリトライしない）

すべての重複窓を閉じる代わりに喪失窓を開く。即不採用: 確実な配送は outbox の存在意義である。

### 受信側検証つき fencing token

end-to-end で重複を閉じられるが、出荷済み契約を超えた受信側の協力を要する。すなわち吸収と同じ
第三者依存に、より強い結合を加えたもの。単独機構としては不採用。

## 備考

- 設計リファレンス: [`docs/design/outbox.md`](../../design/outbox.md) — §1 の不変条件と §4 の
  統合者チェックリストが、本決定が依拠する受信側 dedup 義務を規定する。
- 関連 ADR: [ADR-0031]（tx リトライ冪等性契約。出荷状態の relay はその唯一の公認例外であり、
  設計図の第 2 層がこの例外を除去する）、[ADR-0051]（at-least-once poll）、[ADR-0052]
  （SKIP LOCKED claim）、[ADR-0053]（message-id / Idempotency-Key 伝搬）、[ADR-0054]
  （max attempts 到達での dead 化）。
- ADR 全体の一覧と順序: [ADR ログ](README.ja.md)。

[ADR-0031]: 0031-transaction-retry-idempotent-callers.ja.md
[ADR-0051]: 0051-at-least-once-outbox-poll.ja.md
[ADR-0052]: 0052-skip-locked-outbox-relay.ja.md
[ADR-0053]: 0053-message-id-idempotency-propagation.ja.md
[ADR-0054]: 0054-outbox-dead-after-max-attempts.ja.md
[ADR-0100]: 0100-no-in-app-rate-limiter.ja.md
[ADR-0101]: 0101-scheduled-job-concurrency-delegated.ja.md
