# go-boilerplate

Golang × Echo × OpenAPI × PostgreSQL × Onion Architecture によるベースプロジェクトです。
`uber/fx` による DI や `sqlc`, `golang-migrate`, `oapi-codegen` などを採用しています。

## 利用ツール(サポートバージョン)

- Go(1.24.4)
- Docker Desktop
- Github CLI

## 構成スタック

- **言語**: Go
- **Webフレームワーク**: Echo
- **DI**: uber/fx
- **API定義**: OpenAPI
  - **コード生成**: oapi-codegen
- **DB**: PostgreSQL
- **ORM/Query**: sqlc
- **マイグレーション**: golang-migrate
  - **マイグレーション統合**: tern
- **開発補助**:
  - godotenv
  - zap
  - testify
  - cobra（CLI）
  - air（ホットリロード）
  - Docker / docker-compose

## ディレクトリ構成

<details>
<summary>展開する</summary>

```mermaid
graph TD
  A[your-boilerplate] --> B[cmd/]
  B --> B1[server/]
  B1 --> B1a[main.go]

  A --> C[internal/]
  C --> C1[config/]
  C1 --> C1a[config.go]

  C --> C2[domain/]
  C2 --> C2a[user/]
  C2a --> C2a1[entity.go]
  C2a --> C2a2[repository.go]

  C --> C3[infrastructure/]
  C3 --> C3a[postgres/]
  C3a --> C3a1[queries.sql.go]

  C --> C4[interface/handler/]
  C4 --> C4a[user_handler.go]

  A --> D[database/]
  D --> D1[migrations/]
  D1 --> D1a[V01__create_user_table.sql]

  A --> E[openapi/]
  E --> E1[api.yaml]

  A --> F[test/integration/]
  F --> F1[user_handler_test.go]

  A --> G[.vscode/]
  G --> G1[settings.json]
```

</details>

---

## 開発開始手順

```bash

```
