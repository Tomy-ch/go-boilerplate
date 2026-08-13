# 設計リファレンス

English: [README.md](README.md)

本ディレクトリは **サブシステム別の設計リファレンス**を収めています。各文書は 1 つのサブシステムの *役割論・状態遷移 / ライフサイクル・実装箇所・integrator が書く箇所・用語* を、**実装を精査して 1 枚に**まとめたものです。

これらはパッケージの `README` を置き換えるものではなく**補完**します。README は 1 パッケージの API / 概要、設計リファレンスはサブシステム全体を跨ぐ物語です。設計判断の ADR は [`docs/adr/`](../adr/README.ja.md) にあります。

## 読み方

すべてを読む必要は**ありません**。各サブシステムは独立して採用でき、実際に使うサブシステムの
リファレンスだけを読めば十分です。

すべての文書に共通する 2 つの不変条件があります。

- **REST / Job / Worker は Usecase 層への *driving adapter* であり、Usecase / Domain の
  分割軸ではありません。** Usecase / Domain は業務能力・業務文脈・トランザクション境界で
  分割し、transport(REST / Job / Worker)では分割しません。
- **レイヤ境界は設計方針に留めず、`golangci-lint` の depguard により CI で強制されます。**
  cross-layer import(例: `domain` が `infrastructure` を import）は CI で落ちます。設計
  リファレンスは境界が存在する *理由* を説明し、linter がその遵守を保証します。

## 文書一覧

| 文書 | サブシステム | 対象 | パッケージ README |
| --- | --- | --- | --- |
| [rest.ja.md](rest.ja.md) | REST (HTTP) scaffold | 同期リクエストの入口：handler 実装・routing・エラーマッピング | [controller](../../internal/controller/README.ja.md) |
| [worker.ja.md](worker.ja.md) | Worker scaffold | pull-ack queue の入口：engine・seam・Ack/Nack・circuit・drain | [worker](../../internal/controller/worker/README.ja.md) |
| [job.ja.md](job.ja.md) | Job scaffold | CLI / スケジュール実行の入口と状態遷移 | [job](../../internal/controller/job/README.ja.md) |
| [outbox.ja.md](outbox.ja.md) | Transactional outbox | outbox パターンによる信頼性のあるイベント送出 | [outbox](../../internal/usecase/boundary/outbox/README.ja.md) |
| [idempotency.ja.md](idempotency.ja.md) | Idempotency | `Idempotency-Key` サブシステムと GC ジョブ | [idempotency](../../internal/usecase/idempotency/README.ja.md) |
| [observability.ja.md](observability.ja.md) | Observability | 横断的な traces / metrics / logs 基盤 | [observability](../../internal/observability/README.ja.md) |
| [auth.ja.md](auth.ja.md) | 認証 | RS 側の JWT / JWKS 検証と開発用 OIDC provider（`mock-auth-server`） | [jwt](../../internal/infrastructure/auth/jwt/README.ja.md) |
| [security.ja.md](security.ja.md) | セキュリティ姿勢 | 脅威モデル、各制御が何のためにあるか（強制 / 検知 / 抑止）、どこで発火するか | [workflows](../../.github/workflows/README.ja.md) |
| [context-map.ja.md](context-map.ja.md) | コンテキストマップ | このシステムが周囲のシステムとどう関係しているかを辺ごとに | [boundary](../../internal/usecase/boundary/README.ja.md) |
| [agent-environment.ja.md](agent-environment.ja.md) | エージェント環境 | 指示、機械的 gate、独立 review、負荷を考慮した検証がどう協調するか | [AGENTS.md](../../AGENTS.md) |

## 読む順序

各文書は独立していますが、**入口 → 信頼性サブシステム → 横断**の順で自然に読めます：

1. **入口** — [rest](rest.ja.md)（同期）, [worker](worker.ja.md)（非同期）, [job](job.ja.md)（CLI / スケジュール）
2. **信頼性サブシステム** — [outbox](outbox.ja.md), [idempotency](idempotency.ja.md)
3. **横断** — [observability](observability.ja.md), [auth](auth.ja.md), [security](security.ja.md)
