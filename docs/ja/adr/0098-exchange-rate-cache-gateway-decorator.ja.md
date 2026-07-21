---
status: accepted
date: 2026-07-22
deciders: [maintainers]
tags: [architecture, http]
---

# ADR-0098: 為替レート gateway を boundary 継ぎ目上の TTL decorator でキャッシュする

English canonical: [0098-exchange-rate-cache-gateway-decorator.md](../../adr/0098-exchange-rate-cache-gateway-decorator.md)

## ステータス

accepted

## 背景

`GET /v1/exchange-rates` は、外部サービス（Frankfurter、ECB 由来）から `boundary.Gateway` 継ぎ目を通じてレートを取得する。レート源の更新は日次が上限であり、リクエストごとに外部 HTTP 呼び出しを繰り返すのは無駄で、リクエスト遅延を外部依存に結びつける。backend キャッシュが必要である。

[ADR-0096](0096-no-generic-cache-abstraction.ja.md) はすでに汎用 `Cache` 抽象を否定し、既存継ぎ目上の **decorator** を規定している。ただしその決定文はドメイン `Repository` インターフェースを swap 継ぎ目の主語としている。本改修のキャッシュ対象は `internal/usecase/boundary/exchangerate.Gateway`（外向き gateway boundary）であり `Repository` ではない。したがって、(a) ADR-0096 の Repository 主語を踏まえたキャッシュの置き場、(b) TTL 値とそれが含意する鮮度許容、の判断が必要である。

## 決定

為替レート gateway を、`boundary.Gateway` を満たす in-memory TTL **decorator** でキャッシュする。decorator は infra 層（`internal/infrastructure/webapi/exchangerate/cache.go`）に置き、DI（`internal/di/module/webapi.go`）で素の gateway を包んでから usecase へ注入する。usecase / domain はキャッシュを意識しない。

ADR-0096 の decorator 原理——*汎用抽象ではなく既存の内層継ぎ目上の decorator でキャッシュする*——を、`Repository` に限らず `Gateway` boundary にも適用すると明示的に読み替える。主語は異なるが継ぎ目の原理は同一である。

TTL は `const rateTTL = 24 * time.Hour` の固定値とし、env / config 値にはしない。レート源が日次更新である以上、24h は根拠を説明できる値である。ECB 公表時刻（≈16:00 CET）を跨ぐと最大 ~24h 古いレートを返しうるが、キャッシュ値は**非永続の参考表示**にのみ供され、`rate_date` をレスポンスに露出するため利用側が鮮度を判断できる。よって許容する。

## 結果

### ポジティブな結果

- usecase / domain はキャッシュを意識しない。TTL の時刻依存は infra decorator に完全に閉じる（onion、[ADR-0002](0002-onion-architecture.ja.md) と整合）。
- [ADR-0096](0096-no-generic-cache-abstraction.ja.md) と整合: 汎用キャッシュ抽象を持たず、既存継ぎ目を再利用する。
- 同一通貨ペアの反復要求に対する外部 HTTP 呼び出しを削減する。

### ネガティブな結果

- ADR-0096 の Gateway boundary 読み替えは Repository 主語の文面の拡張であり、その理由をここに記録して暗黙化を避ける。
- 固定 24h TTL は日次公表境界を跨いで古いレートを返しうる。`rate_date` の露出と参考値である点で緩和する。

### 中立的な結果

- エラーはキャッシュしない。取得失敗は次回要求で再度レート源に到達するため、一時障害がキャッシュを汚染しない。

## 検討した代替案

### usecase 内 in-memory キャッシュ

却下: usecase が実時刻と可変状態に依存し、clock 抽象が必要になり onion 境界を濁す。キャッシュは infra の関心事である。

### 専用のキャッシュ boundary インターフェース

[ADR-0096](0096-no-generic-cache-abstraction.ja.md) が明示的に否定する汎用キャッシュ抽象を再導入するため却下。

### TTL を config 値にする

却下: 環境ごとに変える理由がなく、「根拠を言えない値を置かない」という config 原則に反する。具体的な運用要件が生じた時点で ADR 改訂とともに config 化すればよい。

## 備考

- 関連: [ADR-0096](0096-no-generic-cache-abstraction.ja.md)（decorator 継ぎ目原理。ここでは `Gateway` boundary へ読み替え）。
- 関連: [ADR-0002](0002-onion-architecture.ja.md)（decorator を infra 層で結線）。
- 本キャッシュが供給するレートの参考額の丸め: [ADR-0099](0099-reference-amount-half-up-rounding.ja.md)。
