# Dockerfile (local 開発環境用)

このDockerfileは、Golang+Echo+SQLC+OpenAPIを中心としたプロジェクトにおける**開発用コンテナ**を定義しています。

ホットリロード(`air`)、デバッガ(`dlv`)、コード整形(`gofumpt`,`golines`)、Linterなど、日常的に開発に利用するツールが揃っています。

## 利用目的

- Alpineベースの軽量なGo開発環境
- `make`/`air`/`golangci-lint`/`sqlc`などを手元で実行

## 事前インストールされるツール一覧

### OSレベル (apk add)

| パッケージ       | 目的・用途        |
|------------------|-------------------------------------------------------------------|
| `build-base`     | `golangci-lint`などで利用。`gcc`/`make`/`libc-dev`など、C依存ビルドのためのツール群|
| `binutils-gold`  | `golangci-lint`などで利用。高速リンカ (`ld.gold`)。ビルド高速化目的 |
| `bash`           | 開発中のスクリプト、Makefile実行などに必要                             |
| `curl`           | APIテスト・ファイル取得用                                             |
| `git`            | `go install`や`go mod`の依存管理で必要                         |
| `upx`            | GoバイナリをUPX圧縮する場合（任意）                                   |
| `libc6-compat`   | Alpine上でglibcバイナリを動かす互換レイヤー                             |
| `gcompat`        | glibc互換性対応の補完                                             |
| `tzdata`         | JST/UTCタイムゾーンサポート（ログ表示の整合性確保）                       |
| `make`           | Makefile 実行用                                               |

### Go ツールチェーン (`go install`)

| ツール名                   | 説明 |
|----------------------------|------------------------------------------------------------|
| `air`                      | ホットリロードツール                                      |
| `dlv`                      | Delve: Goのデバッガ                                   |
| `golines`                 | 行長制限つきのGo整形ツール（`gofumpt` ベース）             |
| `lefthook`                 | Git hook 管理ツール（test/lint自動実行に利用）             |
| `gofumpt`                  | `gofmt` + alpha の整形強化版                           |
| `gomodifytags`             | 構造体のタグ編集 CLI ツール                                |
| `golangci-lint`            | 多機能Go linter（`make lint` などで使用）                 |

## ワークディレクトリ

```Dockerfile
WORKDIR /app
```

プロジェクトルートを`/app`に指定しています。

## デフォルトCMD

```Dockerfile
CMD ["air", "-c", ".air.toml"]
```

コンテナ起動時に`air`によるホットリロードを自動実行します。

## Tips

- 実行例：

```sh
make serve      # air 経由でアプリ起動
make lint       # golangci-lint 実行
make test       # go test 実行
```

## 参考リンク

- [air](https://github.com/cosmtrek/air)
- [golangci-lint](https://golangci-lint.run/)
- [oapi-codegen](https://github.com/deepmap/oapi-codegen)
- [swagger-cli](https://www.npmjs.com/package/swagger-cli)
