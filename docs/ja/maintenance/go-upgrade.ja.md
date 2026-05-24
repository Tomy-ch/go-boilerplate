# Go Version Upgrade Procedure

このドキュメントは、このプロジェクトで **Go のバージョンを更新する際の手順**を説明します。

Go のバージョン更新は以下に影響します。

- Go toolchain
- 依存パッケージ
- Go tools
- Code generation
- CI
- Docker images

そのため、以下の手順に従って更新してください。

## 1. Release Notes の確認

Go programming language

対象バージョンの Release Notes を確認します。

確認する主な項目

- language spec の変更
- standard library の breaking change
- `go vet` の変更
- toolchain の変更

例

```text
https://go.dev/doc/devel/release
```

## 2. `.go-version` を更新

このプロジェクトでは Go バージョンを `.go-version` で管理しています。

```text
.go-version
```

このファイルの内容を更新してください。

```text
1.26.2
```

## 3. ローカル Go 環境の更新

このプロジェクトでは **goenv の利用を推奨しています（必須ではありません）**。

### goenv を使用する場合

```sh
make go-update
```

確認

```sh
go version
```

### goenv を使用しない場合

Homebrew を利用している場合

```sh
brew update
brew upgrade go
```

確認

```sh
go version
```

## 4. CI の Go version 更新

GitHub Actions の Go version を更新します。

対象ディレクトリ

```text
.github/workflows
```

例

```yaml
- uses: actions/setup-go@v6
  with:
    go-version-file: go.mod
    cache: true
```

## 5. `go.mod` の Go version 更新

```sh
go mod edit -go=1.26.2
```

## 6. 依存関係と vendor の更新

このプロジェクトでは依存関係の整理に **Makefile タスク `tidy-lib`** を使用します。

```sh
make tidy-lib
```

このタスクは以下を実行します。

- `go mod tidy`
- `go mod vendor`

## 7. Go tools の再インストール

Go version を更新した場合、Go tools は古い Go version でビルドされたバイナリのままになります。

そのためツールを再インストールしてください。

```sh
make install-tools
```

インストールされる主なツール

- gopls
- golangci-lint
- delve
- lefthook
- gotests
- impl
- goplay

## 8. Docker イメージの更新

Dockerfile の Go version を更新します。

例

```dockerfile
FROM golang:1.26.2
```

## 9. Docker コンテナ再ビルド

Go バージョンアップでは base image タグが変わるため、新しいイメージを確実に pull・再ビルドできるよう `-clean`（`--no-cache --pull`）バリアントを使います。

サーバー系コンテナ

```sh
make serve-build-clean
```

ツール系コンテナ

```sh
make tools-build-clean
```

## 10. Code generation の再実行

Go version の変更により生成コードが変わる可能性があります。

```sh
make gen
```

## 11. テスト実行

```sh
make test
```

または

```sh
go test ./...
```

## 12. Lint 実行

```sh
make lint
```

## 13. 最終確認

以下のコマンドがすべて成功することを確認してください。

```sh
make tidy-lib
make install-tools
make gen
make test
make lint
make serve-build-clean
make tools-build-clean
```

## Upgrade Checklist

Go version を更新する際は以下を確認してください。

- [ ] Release Notes 確認
- [ ] `.go-version` 更新
- [ ] ローカル Go 更新
- [ ] CI Go version 更新
- [ ] `go.mod` Go version 更新
- [ ] `make tidy-lib` 実行
- [ ] `make install-tools` 実行
- [ ] Dockerfile 更新
- [ ] Docker コンテナ再ビルド
- [ ] code generation 再実行
- [ ] test 実行
- [ ] lint 実行
