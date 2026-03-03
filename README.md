# go-boilerplate

Golang × Echo × OpenAPI × PostgreSQL × Onion Architecture によるベースプロジェクトです。

`uber/fx` による DI、`sqlc`, `golang-migrate`, `oapi-codegen` を採用し、  
**契約駆動 + 型安全 + レイヤ分離** を前提とした構成になっています。

## Architecture Overview

本プロジェクトは **軽量 Onion Architecture** を採用しています。

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

## Intended Use Cases

このBoilerplateは以下の用途を想定しています：

- 新規プロダクトのバックエンド構築  
- PoC〜初期スケールフェーズ  
- 厳格なレイヤ分離が必要なチーム開発  
- 型安全なSQL管理が必要なプロジェクト  
- AI支援開発を前提とした設計統制  

以下の用途には向きません：

- 単一ファイルで完結する小規模API  
- アーキテクチャ境界を設けない高速プロトタイピング  

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

- レイヤ強制
- 生成物分離
- release基準ブランチ運用
- OpenAPI-first
- domain純粋性の維持

これらはアーキテクチャの逸脱や意図しない変更を防ぐための制約です。

## 本テンプレートの前提

本テンプレートは以下を理解しているチーム向けです：

- Go + Echo + Fx + OpenAPI + sqlc 構成
- レイヤアーキテクチャ
- .env 管理とセキュリティ境界の判断
- 初期構築の意思決定が可能（TL相当）

## Tools (Required Versions)

- Go
  - バージョンは `go.mod` を参照
- Docker Desktop
- GitHub CLI
- Postman

## License

This project is licensed under the MIT License.
See [LICENSE](./LICENSE) for details.

## 参考

- [バージョニングルール](docs/versioning.md)

This repository is not just a collection of libraries.
Its primary value lies in architectural constraints and design philosophy.
