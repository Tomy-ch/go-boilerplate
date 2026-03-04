# go-boilerplate

[English](README.md) | 日本語

Golang × Echo × OpenAPI × PostgreSQL × Onion Architecture によるベースプロジェクトです。

`uber/fx` による DI、`sqlc`, `golang-migrate`, `oapi-codegen` を採用し、**契約駆動 + 型安全 + レイヤ分離** を前提とした構成になっています。

## Why This Boilerplate Exists

バックエンド開発では、プロジェクトごとに

- アーキテクチャ
- ライブラリ選定
- ディレクトリ構成
- 開発フロー

を一から設計する必要があります。

その結果、同じ設計議論や試行錯誤が繰り返されることが少なくありません。

このBoilerplateは、そうした **初期設計コストを削減し、安全に開発を開始できるベースライン** を提供するために作成されました。

本テンプレートは

- Onion Architecture
- OpenAPI-first
- sqlcによる型安全なSQL
- Dependency Injection
- CIによる構造チェック

を組み合わせた **契約駆動・型安全・レイヤ分離** のバックエンド構成を提供します。

このBoilerplateの価値は、特定のライブラリではなく **広く使われているOSSを統合した設計そのもの** にあります。

## Architecture Overview

本プロジェクトは **Onion Architecture** を採用しています。

`controller → usecase → domain ← infrastructure`

- 依存関係は必ず内側へ向かう
- domain は純粋で副作用を持たない
- infrastructure は domain の interface を実装する
- controller はビジネスロジックを持たない

## API Development Policy (OpenAPI First)

本プロジェクトは **OpenAPI-first** で開発します。

API変更は必ず以下の順序で行います：

1. `openapi/` の定義を修正  
2. `make gen-api` でコード生成  
3. handler / usecase を実装  

生成ファイルは手動で編集してはいけません。

## Branch Strategy

本リポジトリは **release-centric branching model** を採用しています。

- 機能ブランチは最新の `release/*` から作成する  
- `develop`, `staging`, `production` へは release 経由でのみ反映される  
- 保護ブランチへの直接コミットは禁止  
- すべての変更は Pull Request 経由で行う  

この戦略により：

- バージョン整合性の維持  
- 安全なリリースフロー  
- AI支援開発時の事故防止  

を実現しています。

## Directory Structure

```text
.
├── cmd/            # Application entrypoint
├── internal/       # Application code (Onion Architecture)
│   ├── domain/
│   ├── usecase/
│   ├── infrastructure/
│   ├── controller/
│   └── di/
├── database/       # Migrations & SQL (sqlc)
├── openapi/        # API contracts (OpenAPI-first)
├── pkg/            # Shared utilities
├── docker/
├── docs/
└── makefile
```

詳細な構造説明は各ディレクトリ直下の README を参照してください。

## Stack

- **Language**: Go
- **Web Framework**: Echo
- **DI**: uber/fx
- **API Definition**: OpenAPI + oapi-codegen
- **DB**: PostgreSQL
- **Query**: sqlc
- **Migration**: golang-migrate (+ tern)
- **Logging**: zap
- **Testing**: testify
- **CLI**: cobra
- **Dev Tools**: Docker / docker-compose / air

## Getting Started

```bash
make install
make serve
make tools
make db-init
```

## Release Workflow

タグ発行：

```bash
make release-major-tag # メジャーリリース
make release-minor-tag # マイナーリリース
make release-patch-tag # パッチリリース
```

次リリースブランチ作成：

```bash
make release-major-branch
make release-minor-branch
make release-patch-branch
make hotfix-patch-branch
```

## AI-Safe Design

本テンプレートは AI支援開発を前提に設計されています。
これらは AIエージェントによるコード生成時の事故を防ぐための設計でもあります。

- レイヤ強制
- 生成物分離
- release基準ブランチ運用
- OpenAPI-first
- domain純粋性の維持

これらはアーキテクチャの逸脱や意図しない変更を防ぐための制約です。

## Tools (Required Versions)

- Go
  - バージョンは `go.mod` を参照
- Docker Desktop
- GitHub CLI
- Postman

## Architecture Philosophy

このテンプレートは **強いアーキテクチャ方針** を持っています。

そのため、すべてのプロジェクトに適しているわけではありません。  
要件やチーム構成によっては、別のテンプレートを利用するか  
フォークしてカスタマイズすることを推奨します。

一方で、このBoilerplate自体は継続的に改善していきたいと考えています。

設計に関する議論や改善提案は歓迎していますので、  
気になる点やアイデアがあれば Issue や Discussion で気軽に共有してください。

このBoilerplateは **実行速度よりも保守性と堅牢性を優先**しています。

主な特徴：

- Onion Architecture によるレイヤ分離
- OpenAPI-first による契約駆動開発
- sqlc による型安全なSQL
- DIによる疎結合な設計
- CI / Lint / Code Generation による構造的な品質保証

このプロジェクトでは、

- 善意のコードレビュー
- 暗黙のルール
- 属人的な判断

に過度に依存しない設計を重視しています。

その代わりに、**構造・生成コード・CIによる機械的な安全性の確保**を重視しています。

詳しくは、[architecture.md](./docs/architecture.md) を参照してください。

## Assumed team

本テンプレートは以下を理解しているチーム向けです：

- Go + Echo + Fx + OpenAPI + sqlc 構成
- OpenAPI を用いた契約駆動開発
- SQL を中心としたデータアクセス設計
- レイヤアーキテクチャ（Onion / Clean Architecture 等）
- Docker / Docker Compose を用いた開発環境
- .env 管理とセキュリティ境界の判断
- 初期構築の意思決定が可能（TL相当）

## Intended System Types

このテンプレートは以下の用途を想定しています。

- 新規プロダクトのバックエンド構築
- PoC〜初期スケールフェーズ
- 厳格なレイヤ分離が必要なチーム開発
- 型安全なSQL管理が必要なプロジェクト  
- ドメイン制約（法律・商習慣など）が多いシステム
- ある程度の長期運用を前提としたバックエンド
- ドメイン制約の多い業務システム

以下の用途には向いていない可能性があります。

- 単一ファイルで完結する小規模API
- アーキテクチャ境界を設けない高速プロトタイピング
- 極端に低レイテンシを要求されるシステム
- 強いマイクロサービス分割を前提とするシステム

本テンプレートは **Onion Architecture を採用したモジュラーモノリス** を前提に設計されています。

## SaaS / Vendor Neutrality

Observabilityなどについては  
Datadogなど特定のSaaSに依存しないように設計しています。

そのため、

- OSSベースのツール
- ベンダーニュートラルな設計

を基本方針としています。

## Extensibility

`internal/` 配下のコードは可能な限り疎結合になるよう設計されており、  
DIによってコンポーネントを配線しています。

そのため、

- インフラ
- 実装
- ミドルウェア

などは環境に応じて差し替え可能です。

## Maintainer Policy / Disclaimer

本リポジトリは **作者個人によって作成・公開されているプロジェクト**です。  
特定の企業・団体・組織に依存するものではなく、設計や実装方針は作者個人の思想に基づいています。

本テンプレートは善意に基づいて公開されていますが、  
**特定用途への適合性、セキュリティ、運用上の安定性を保証するものではありません。**

本テンプレートを利用する場合は、  
利用者自身の責任で以下を確認した上で使用してください。

- 依存ライブラリの脆弱性
- セキュリティ設定
- 運用要件との整合性

## Library Selection Policy

このBoilerplateの価値は、特定のライブラリそのものではなく  
**広く利用されているOSSを適切に統合している点**にあります。

依存ライブラリは原則として以下の基準で選定しています。

- 活発なコミュニティによって継続的にメンテナンスされていること
- 個人ライブラリへの過度な依存を避けること
- 将来的に差し替えやforkが可能であること
- 特定フレームワークへのロックインを避けること

一部例外はありますが、基本的には **置き換え可能な設計**を前提としています。
必要に応じて fork や差し替えがでの対応が可能なライブラリを選定しています。

## Maintenance Policy

作者は可能な範囲で以下を実施します。

- 依存ライブラリの更新
- セキュリティアップデート
- アーキテクチャ改善

ただし、

- Issueの対応期限
- バグ修正の保証
- 長期的なサポート

については **保証できません**。

バグや問題を見つけた場合は Issue で報告していただけると助かります。  
可能な範囲で対応します。

## Future Boilerplates

今後以下のBoilerplateも公開予定です。

- Frontend Boilerplate
- Infrastructure Boilerplate
- Observability Boilerplate

公開された場合は本リポジトリからリンクします。

## AI-Agent Documentation

このリポジトリには AIエージェント向けドキュメントも含まれています。

ただし、**AIを使用しなくても人間だけで運用できるように必要なドキュメントはすべて整備する方針です。**

## License

このプロジェクトはMITライセンスの下で提供されています。
詳細は[LICENSE](./LICENSE)をご覧ください。

## Reference

- [バージョニングルール](docs/project/ja/versioning.ja.md)
- [アーキテクチャ](docs/ja/index.ja.md)
  - [アーキテクチャドキュメント](docs/ja/architecture.ja.md)
  - [開発フロー](docs/ja/development-flow.ja.md)
  - [設計判断の背景](docs/ja/decisions.ja.md)
  - [アーキテクチャルール](docs/ja/rules.ja.md)
- [go/ライブラリアップデート方法](docs/maintenance/ja/go-upgrade.ja.md)
