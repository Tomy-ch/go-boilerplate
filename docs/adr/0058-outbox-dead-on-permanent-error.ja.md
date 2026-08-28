---
status: accepted
date: 2026-08-28
deciders: [maintainers]
tags: [outbox, async, reliability]
---

# ADR-0058: outbox 行は恒久的なエラーで dead になり、一時的な失敗は message ごとの backoff で retry する

## ステータス

accepted

## 背景

一部の outbox 行は、何回 retry しても成功しない——恒久的に設定を誤った endpoint、receiver が不正として
拒否する payload、廃止された receiver。終端状態が無ければこれらの行は永遠に `pending` のままで、poll の
たびに relay の容量を消費し、本物の配送問題を隠すノイズで lag metric を汚す。

この決定の最初の版は、終端状態を *回数* の関数にしていた: 10 回失敗したら `dead`。その規則は message ごとの
backoff を持たない relay の安全弁だった——失敗した行は次の poll で即座に再 claim されるので、*何か* が
止めなければならなかった——そして、message を拒否する receiver と一時的に落ちている receiver を区別
することは決してなかった。[ADR-0111] がその帰結を記録している: 数十秒の downstream 障害が pending の
backlog 全体を `dead` へ追いやり、回復は operator が全行を手で replay することになる。

realtime 配送 channel（[ADR-0072]）は、その帰結を不便から correctness の問題に変える。その順序規則は
まだ append されていない最初の sequence で stream を止める。一時的な障害がその先頭行を dead にできる
なら、1 分の substrate の不調が operator の介入まで全 active stream を止める。さらにこの channel には
message ごとの恒久失敗がそもそも無い——payload は行を書く前に検証され、substrate は append を受け付ける
か全員に対して到達不能かのどちらかである——ので、回数基準の規則がそこで発火するのは一時的な失敗に
対してだけになる。

## 決定

**行が `dead` になるのは失敗が恒久的だからであり、頻繁に失敗したからではない。**

- publisher は `apperror` の sentinel（[ADR-0047]、[ADR-0048]）で分類した error を返す: retry しても
  変わらない失敗は `ErrPermanent`、変わり得る失敗は `ErrRetryable`。HTTP publisher は outbound client の
  既存の判定（[ADR-0024]）から class を導く: retry 不可の応答（429 以外の 4xx）は permanent、5xx / 429 /
  transport 失敗は retryable。realtime publisher は到達不能な substrate を retryable として報告する。
  payload 検証は emit 前に行うので、permanent class を持たない。
- **permanent → 即座に `MarkDead`。** `last_error` に理由を記録し、`outbox.dead` を増やし、warning を
  log する。dead 行は operator が `outbox-relay replay [--message-id=<uuid>]` を実行するまで終端で
  ある——これまでどおり。
- **retryable → 行は `pending` のまま、`next_attempt_at` を進める。** full jitter 付きの指数 backoff、
  上限 60 秒。claim 述語に `next_attempt_at <= now()` を加えるので、backoff 中の行は単に選ばれない
  ——lock されることがなく、だからこの決定の最初の版が恐れた複雑さ無しに `FOR UPDATE SKIP LOCKED`
  （[ADR-0056]）と共存する。`attempts` は診断のために増え続けるが、もはや何の基準でもない。
- **どちらの sentinel も持たない error は retryable として扱う。** worker engine が未分類の error に
  適用しているのと同じ既定であり、message を失わない側に倒す。手放すのは、誰も分類しなかった恒久失敗
  に対する旧来の回数基準の backstop である。それは counter ではなく観測で補う: delivery channel ごとの
  最古の `pending` 行の age を lag SLI とし、閾値を超えて pending に留まる行は alert する——数字で
  dead にするのではなく、人間が分類できるように表面化させる。

status 値は `pending`、`published`、`dead` の 3 つのまま。`failed` も `backing-off` も無い——backoff は
pending 行の timestamp である。

実装は親 issue の outbox delivery-channel の作業（`next_attempt_at` を導入するのと同じ変更）で入る。
`docs/design/outbox.md` はその変更で更新する。

## 影響

### ポジティブな影響

- downstream の障害が費やすのは latency であって dead な backlog ではない。行は待ち、間隔を広げながら
  retry し、receiver が戻れば流れる。relay はもはや一時的な失敗を incident に変える部品ではない。
- 恒久失敗は、何も足さない同じ試行を 10 回重ねた後ではなく、最初の発生で理由付きの dead になる。
- realtime channel の head-of-line 規則を安全に保てる。stream が止まるのは、この channel が生まない
  恒久失敗か、自然に解消する障害のときだけである。
- relay が使う分類は codebase の他の部分が既に話しているものであり、新しい error 分類法を持ち込まない。

### ネガティブな影響

- 誰も分類しなかった恒久失敗——永遠に `500` を返す receiver、transport error として現れる bug——は
  backoff 上限で無期限に retry する。自動で終端させるものは無く、lag SLI とその alert だけが守りで
  あり、それは監視されなければならない。
- `next_attempt_at` 列と claim 述語の変更は schema migration と index の変更であり、設定の切り替え
  ではない。
- backoff の parameter（初期間隔、上限、jitter）は code 上の固定値である。違う値が欲しい deployment は
  code を変える。

## 検討した代替案

### 固定回数（`MaxAttempts = 10`）で dead——以前の決定

置き換えた。message ごとの backoff を持たない relay の安全弁であり、失敗の種類ではなく頻度で dead に
する: 30 秒落ちている receiver と決してその message を受け付けない receiver が、それには同じに見える。
realtime channel の順序規則の下では、短い障害で全 active stream を止めることになる。

### 分類も backoff も無い無制限 retry

これまでどおり却下: 配送不能な行が蓄積し、毎 cycle の poll 容量を消費し、本物の lag を隠す。
「attempt 上限なし」を安全にするのは分類である——恒久失敗は最初の出現で loop を離れる。

### 使い果たしたら削除

これまでどおり却下: data が失われ回復経路が無い。

### 未分類の error を permanent として扱う

却下。publisher が予見しなかった全失敗を dead にする。そのほとんどは一時的であり、この決定が取り除く
挙動そのものである。retryable の既定 + age alert は何も失わず、稀な恒久ケースを人間へ表面化させる。

### backstop として非常に大きな attempt 上限を残す

却下。1000 という上限も分類を装った回数であり、根拠のある値が無く、規模を変えて障害→dead の経路を
再現する。

## 備考

- 設計正本: `docs/design/outbox.md`（§「State transitions」、用語集の「dead」と「backoff」）。
- 本決定は [ADR-0111] の hardening blueprint の項 3（`next_attempt_at` による message ごとの backoff）
  を単独で採択する——運用エビデンスによってではなく、realtime channel が回数基準の dead を許容できない
  からである。blueprint の他の項はその ADR が述べるとおり延期のまま。
- 関連: [ADR-0024]（HTTP publisher が写像する outbound client の retryable / non-retryable 判定）、
  [ADR-0047] / [ADR-0048]（sentinel）、[ADR-0055]（poll による retry）、[ADR-0056]（claim 述語）、
  [ADR-0059]（retention GC）、[ADR-0072]（dead な先頭行が stream を止める理由）。

[ADR-0024]: 0024-outbound-http-resilience.ja.md
[ADR-0047]: 0047-apperror-protocol-agnostic-errors.ja.md
[ADR-0048]: 0048-error-metadata-code-message-details.ja.md
[ADR-0055]: 0055-at-least-once-outbox-poll.ja.md
[ADR-0056]: 0056-skip-locked-outbox-relay.ja.md
[ADR-0059]: 0059-outbox-retention-gc.ja.md
[ADR-0072]: 0072-postgres-state-dynamodb-eventlog.ja.md
[ADR-0111]: 0111-outbox-relay-hardening-delegated.ja.md
