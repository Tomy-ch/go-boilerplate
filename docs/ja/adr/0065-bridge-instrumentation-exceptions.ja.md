---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [dependencies, observability, exception]
---

# ADR-0065: ブリッジ / 計装ライブラリを有界な SRP 例外として認める

English canonical: [0065-bridge-instrumentation-exceptions.md](../../adr/0065-bridge-instrumentation-exceptions.md)

## ステータス

accepted

## 背景

ライブラリ選定ポリシー（[ADR-0064](0064-library-selection-policy.ja.md)）は、責任が単一の
上流に結びついていることを要求する。計装ライブラリおよびブリッジライブラリは本質的に
**独立してバージョン管理される 2 つの上流**（フレームワーク / ライブラリ × OpenTelemetry）の間に
立つため、この基準を満たせない。しかし、これらのグルーを手で実装すると対象の内部ライフサイクルに
強く結合し、保守負債を増やすことになる。

## 決定

ブリッジ / 計装ライブラリを単一責任ポリシーへの**明示的かつ個別に正当化された例外**として
認める。共通の根拠は以下の通り。

- グルーを手で実装すると対象（Echo / pgx / zap）の内部ライフサイクルに強く結合し、
  保守負債を削減するどころか増やすことになる。
- それぞれ**小規模かつ Apache-2.0 ライセンス**であるため、最悪の場合はベンダー化 / フォークが
  可能。フォークコストはライブラリごとに記録された本番コード行数の範囲に留まる。
- otel-contrib 系（`otelecho` / `otelhttp` / `otelzap`）は**otel-contrib の月次リリーストレイン**で
  提供され、OpenTelemetry とロックステップで維持される。`otelpgx` は
  サードパーティパッケージ（`github.com/exaring/otelpgx`、Apache-2.0）で、pgx と OpenTelemetry を
  独立に追従する。いずれの場合も残存するドリフトサーフェスはフレームワーク側のインターフェースのみで、
  それら（`echo.MiddlewareFunc` / `net/http.RoundTripper` / pgx `QueryTracer` / `zapcore.Core`）は
  安定した v1 である。

現在受け入れられている例外は `otelecho`（ルートサーバースパン）・`otelhttp`（外向き HTTP クライアント
スパン）・`otelpgx`（SQL クエリスパン）・`otelzap`（zap → OTel ログブリッジ）である。

## 影響

### ポジティブな影響

- 脆いグルーを手で保守することなくオブザーバビリティ計装が得られる。
- 各例外は有界である: ライセンス・上流のリリースサイクル・最悪のフォークコストが既知。

### ネガティブな影響

- これらの依存関係は 2 つの上流をまたぐため、単一上流ライブラリよりドリフトサーフェスが広い
  — これを認識した上で受け入れ、ライブラリごとにレビューする。

## 検討した代替案

### グルーを手で実装する

却下: 各対象の内部ライフサイクルに強く結合し、ブリッジ依存関係より保守負債が増加する。

### 例外を拒否する（計装を断念する）

却下: 有界な最悪フォークコストを考慮すると、標準的かつ低コストのオブザーバビリティ計装を
釣り合わないメリットのために放棄することになる。

## 補足

- 親ポリシー: [ADR-0064](0064-library-selection-policy.ja.md)。関連ゲーティング決定: [ADR-0058](0058-config-driven-observability-gating.ja.md)。
- ライブラリごとのバージョンと行数は調査時点（2026-06-25 調査）のインベントリスナップショットであり、この不変の記録ではなく依存関係リファレンス（`docs/reference/dependencies.md`、Phase 5）に属する。
- 移行元: `docs/decisions.md`（§「Exceptions: instrumentation / bridge libraries」）。
