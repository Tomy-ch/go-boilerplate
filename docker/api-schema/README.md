# API schemaコンテナ

このコンテナは、GoプロジェクトにおけるOpenAPIスキーマを中心としたの**生成・バンドル用**のnodeとgoのコンテナを提供します。

各種ツールでの生成は、docker-composeを通じて行います。

## 利用目的

- `swagger-cli`: OpenAPIのYAMLスキーマのバンドル（`$ref` 解決）
- `redocly/cli`: RedocドキュメントのHTML出力
- `oapi-codegen`: Goコードの自動生成（Echoサーバ/型定義など）
- `mockgen`: Goのinterfaceに基づくMock生成

## 構成

```text
# openapi-builder (Nodeベース)
- swagger-cli
- redocly/cli(swagger-cliで生成されるyamlに依存します)

# go-generator (Goベース)
- oapi-codegen(swagger-cliで生成されるyamlに依存します)
- mockgen
```

## 利用方法

```bash
make gen
```

`go:generate` ディレクティブに従い、`oapi-codegen`や`mockgen`が実行されます。

## 追加方法

新しいコードの生成ツールを追加する場合は、以下の手順に従ってください。

1. 必要なツールをDockerfileにインストール
2. `docker-compose.yaml`に新しいサービスを追加
   1. `profiles`セクションに`generate`を追加
   2. 実行したいコマンドを`command`に指定
3. (必要であれば)`Makefile`に新しいターゲットを追加
   1. 実行がciのみなどで良い場合であれば、`--rm`を用いた一時コンテナ実行を設定

## 生成されるディレクトリ構成

```text
openapi/
├── api.yaml                  # メインのOpenAPI定義
├── components/               # $refで参照されるスキーマ群
├── openapi.gen.yaml          # swagger-cliでバンドルされた出力
└── docs/
    └── index.html            # Redocで生成されたHTML
```

## 注意

- OpenAPI スキーマの`$ref`解決やドキュメント生成は`openapi/`配下が前提です。
- `go generate`に必要なコメントが`internal/interface/handler/...`や`test/...`に記述されている必要があります。

## 関連ツール公式

- [swagger-cli](https://github.com/APIDevTools/swagger-cli)
- [Redocly CLI](https://github.com/Redocly/redocly-cli)
- [oapi-codegen](https://github.com/deepmap/oapi-codegen)
- [mockgen (uber/mock)](https://github.com/uber-go/mock)
