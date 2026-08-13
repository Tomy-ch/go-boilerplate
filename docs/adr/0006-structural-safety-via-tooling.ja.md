---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [foundational, ci, structural-safety]
---

# ADR-0006: ツールと CI で構造的安全性を強制する（depguard）

English canonical: [0006-structural-safety-via-tooling.md](0006-structural-safety-via-tooling.md)

## ステータス

accepted

## 背景

レイヤー依存関係ルールは [`docs/rules.md`](../rules.ja.md) に文書化され、`AGENTS.md` に要約されている。ドキュメントだけでは不十分である: コードレビューの規律に依存するルールは、特にコードベースが成長してコントリビューターが変わるにつれ、一貫性なく適用される。レビューをすり抜けるクロスレイヤーインポート — 例えばユースケースパッケージがインフラストラクチャパッケージをインポートする — は、[ADR-0002](0002-onion-architecture.ja.md) と [ADR-0003](0003-interface-based-decoupling.ja.md) が確立したアーキテクチャ境界を静かに侵食する。

このプロジェクトは、構造的ルールを機械的にチェック可能な制約としてエンコードし、偶発的に迂回されないようにする意図的な選択をした。同じ原則が他の構造的な関心事にも適用される: 生成コードは手動で編集されてはならない（CI の再生成チェックで検証）、API コントラクトは実装より先行しなければならない（OpenAPI ファーストフロー）。

設計目標は保守性・予測可能性・構造的安全性であり、生のパフォーマンスや最小限のツールではない。ツールインフラストラクチャへの投資は長期的な運用性の目標と一貫している。特に重要なのは、同じ CI ゲートが人間が書いたコードと AI が生成したコードの両方に等しく適用されることだ: 境界に違反するエージェントは、人間のコントリビューターと同じビルド失敗を受け取る。

## 決定

`depguard` リンターを使った `golangci-lint` でレイヤー依存関係境界を強制する。禁止されたクロスレイヤーインポートは CI を失敗させる。`.golangci.yaml` に設定される 4 つのコア depguard ルールセットは以下の通り:

- `maintain_a_sound_domain` — ドメインはユースケース・コントローラー・インフラストラクチャパッケージをインポートしてはならない; I/O 側の `pkg/` ユーティリティ（ファイルシステム、プロセス実行、環境変数書き込み）も同様。
- `maintain_a_sound_usecase` — ユースケースはコントローラーやインフラストラクチャパッケージをインポートしてはならない; I/O 側の `pkg/` ユーティリティも同様。
- `maintain_a_sound_controller` — コントローラーはインフラストラクチャパッケージをインポートしてはならない。
- `maintain_a_sound_infrastructure` — インフラストラクチャはコントローラーパッケージをインポートしてはならない。

レイヤーごとに許可・不許可されるものの完全な帰結テーブルは [`docs/rules.md`](../rules.ja.md) にある; この ADR はドキュメントだけでなくツールによってそれらのルールを強制するという決定のみを記録する。

## 影響

### ポジティブな影響

- レイヤー違反はコードレビュー時ではなく CI 時に検出される — 検出は客観的かつ継続的。
- 境界に違反する AI 生成コードは、人間が書いたコードと同じ CI ゲートで失敗する。
- `.golangci.yaml` のリンター設定は、レイヤーごとに許可されたインポートに関する機械可読な権威であり、それが管理するコードと共にバージョン管理される。

### ネガティブな影響

- 制限されたレイヤー内でアクセスすべき正当な新しい `pkg/` ユーティリティを追加するには、`.golangci.yaml` の意識的な更新が必要 — 意図的な摩擦だが、それでも摩擦ではある。
- Depguard はインポートパスで動作する; 禁止されたパッケージをインポートしない単一パッケージ内に留まるアーキテクチャ違反（例: インフラストラクチャファイルに書かれたビジネスロジック）は検出できない。

## 検討した代替案

### ドキュメントとコードレビューのみ

`docs/rules.md` にルールを書き、プルリクエストレビューのみで強制する。却下: レビュアーや時間によって一貫性がなく; AI エージェントには違反検出のための実行時フィードバックループがない。

### レイヤーごとの Go ワークスペースモジュール

各レイヤーを別個の Go モジュールにし、クロスレイヤーインポートを `go build` で失敗させる。より強い分離を提供するが、重大なモジュール管理のオーバーヘッドを追加する（replace ディレクティブ、マルチモジュール CI）。depguard アプローチははるかに低い複雑さで同じ実用的な効果を達成する。

### カスタム go/analysis パス

カスタム静的解析パスを書く。より柔軟だが、専用のリンターを作成・維持する必要がある。Depguard はカスタムコードなしでそのユースケースをカバーする。

## 補足

- 完全なレイヤールールと根拠: [`docs/rules.md`](../rules.ja.md) §§「Layer Dependency Rules」「Usecase Dependency Rules」「Domain Layer Constraints」「Infrastructure Implementation Rules」。
- リンター設定: `.golangci.yaml`（depguard の `rules` ブロック — `maintain_a_sound_domain`、`maintain_a_sound_usecase`、`maintain_a_sound_controller`、`maintain_a_sound_infrastructure`）。
- ソース: `docs/architecture.md` §「Structural Safety」。
- ソース: `docs/rules.md` §「Layer Dependency Rules」（強制注記）。
- 関連: [ADR-0002](0002-onion-architecture.ja.md)（レイヤー形状）、[ADR-0003](0003-interface-based-decoupling.ja.md)（インターフェース境界）。
