# go-boilerplate

Golang × Echo × OpenAPI × PostgreSQL × Onion Architecture によるベースプロジェクトです。
`uber/fx` による DI や `sqlc`, `golang-migrate`, `oapi-codegen` などを採用しています。

## 利用ツール(サポートバージョン)

- Go(1.24.4)
- Docker Desktop
- Github CLI
- Postman

<details>
<summary>手動インストール先のURL</summary>

下記のサイトからダウンロードして進めてください。

- [Golang](https://go.dev/dl/)
- [Docker Desktop](https://docs.docker.com/desktop/setup/install/windows-install/)
- [Github CLI](https://cli.github.com/)
- [Postman](https://www.postman.com/downloads/)

</details>

<details>
<summary>brewでのインストール方法</summary>

コピペで実行できます。

```bash
# anyenvのインストール
brew install anyenv
anyenv init
echo 'eval "$(anyenv init -)"' >> ~/.zprofile

# anyenvのupdateプラグインのインストール
mkdir -p $(anyenv root)/plugins
git clone https://github.com/znz/anyenv-update.git $(anyenv root)/plugins/anyenv-update
anyenv update

# goenvのインストール
anyenv install goenv
goenv install "$(cat .go-version)"

# dockerのインストール
brew install --cask docker

# Github CLIのインストール
brew install gh

# Postmanのインストール
brew install --cask postman
```

</details>

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
