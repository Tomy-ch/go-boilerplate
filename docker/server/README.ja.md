# server Dockerfile

[English](README.md) | 日本語

アプリケーションサーバーの Docker イメージを定義する Dockerfile です。マルチステージビルドにより、本番・マイグレーション・ローカル開発の各ターゲットを提供します。

## ビルドターゲット

|ターゲット|ベースイメージ|用途|
|---|---|---|
|`builder`|`golang:1.26.3-alpine`|Go バイナリのビルド（`ldflags` でバージョン / リビジョン / ビルド日時を埋め込み）|
|`runtime`|`alpine:3.23`|本番実行用コンテナ（非 root ユーザー `app`）|
|`migration`|`runtime` を継承|マイグレーション実行用コンテナ（`migrate-up` コマンド）|
|`tooling`|`golang:1.26.3-alpine`|ローカル開発環境（ホットリロード + デバッグ）|

## runtime

- 非 root ユーザー（`app`）で実行
- `vendor` モードでビルド（`GOPROXY=off`）
- `-ldflags` でバージョン / リビジョン / ビルド日時を埋め込み
- デフォルトコマンド: `./server serve`

## migration

- `runtime` イメージを継承
- `database/migrations` ディレクトリを追加
- デフォルトコマンド: `./server migrate-up`
- アプリケーションデプロイ前に一度だけ実行されるジョブとして使用

## tooling（ローカル開発）

プリインストールされるツール：

|ツール|用途|
|---|---|
|`air`|ホットリロード|
|`dlv`|Go デバッガ|
|`golines`|行長制限付き整形|
|`gofumpt`|gofmt 強化版|
|`golangci-lint`|Go リンター|

OS レベルパッケージ: `build-base`, `binutils-gold`, `bash`, `curl`, `git`, `upx`, `libc6-compat`, `gcompat`, `tzdata`, `make`

デフォルトコマンド: `air -c .air.toml`

## docker-compose サービス

```yaml
api_server:
  dockerfile: docker/server/Dockerfile
  target: tooling
  ports: 8080 (API), 2345 (dlv), 6060 (pprof)
```

## 注意点

- 本番イメージはベースイメージのダイジェストを固定して再現性を確保すること
- `tooling` ターゲットは開発初期の利便性のため `@latest` を使用 — CI 環境ではバージョンを固定すること
- すべてのターゲットで作業ディレクトリは `/app`
