# internal/system

[English](README.md) | 日本語

`internal/system` は **アプリケーションのランタイムメタ情報（ビルド情報）** を提供するパッケージです。

このパッケージはビジネスロジックやインフラとは独立した **プロセス情報 (process metadata)** を扱います。  
主に **バージョン情報・Gitリビジョン・ビルド日時** をアプリケーションから取得できるようにします。

これらの値は通常 **Go build 時の `ldflags` によって注入**されます。

## 役割

このパッケージの責務は次の通りです。

- アプリケーションの **ビルドメタ情報** を保持する
- 実行時に **Version / Revision / BuildDate** を取得できるようにする
- バージョン表示 (`--version`) や診断エンドポイントで利用できる API を提供する
- テスト時に **BuildInfo をモック可能**にする

このパッケージは **ビジネスロジックを持ちません。**

## パッケージ構成

```txt
internal/system
├── buildinfo.go
├── version.go
└── mock/
```

|ファイル|役割|
|---|---|
|`buildinfo.go`|BuildInfo インターフェースと実装|
|`version.go`|ビルド時に埋め込まれるメタ情報|
|`mock/`|テスト用 mock|

## BuildInfo インターフェース

アプリケーションコードは **BuildInfo interface** を通してビルド情報を取得します。

```go
type BuildInfo interface {
 Version() string
 Revision() string
 BuildDate() string
}
```

実装

```go
func NewBuildInfo() BuildInfo
```

利用例

```go
bi := system.NewBuildInfo()

version := bi.Version()
revision := bi.Revision()
buildDate := bi.BuildDate()
```

この設計により

- テスト時に mock を注入可能
- version 取得処理を抽象化
- DI による依存注入が可能

になります。

## version.go

`version.go` は **ビルド時に書き換えられる変数**を定義します。

```go
var (
 Version   = "dev"
 Revision  = "none"
 BuildDate = "2024-12-31T21:00:00Z"
)
```

デフォルト値は **開発環境用のフォールバック値**です。

実際の値は `go build` 時に上書きされます。

## ビルド時の値注入

CI / Docker / Makefile では通常次のように値を注入します。

例：

```bash
go build \
  -ldflags "-X 'internal/system.Version=1.2.3' \
            -X 'internal/system.Revision=abcdef1' \
            -X 'internal/system.BuildDate=2025-01-01T00:00:00Z'"
```

Dockerfile では次のように利用されます。

```bash
ARG VERSION
ARG REVISION
ARG BUILD_DATE

RUN go build \
  -ldflags "-X 'internal/system.Version=$VERSION' \
            -X 'internal/system.Revision=$REVISION' \
            -X 'internal/system.BuildDate=$BUILD_DATE'"
```

## 利用用途

BuildInfo は次の用途で使用されます。

例

```txt
--version コマンド
/version API
/health エンドポイント
ログ出力
診断情報
```

例：

```txt
service version=1.2.3 revision=abc123 build=2025-01-01
```

## レイヤー上の位置

`internal/system` は **アプリケーションのランタイム情報レイヤー**です。

```txt
Controller
Usecase
Domain
Infrastructure
System (runtime metadata)
```

特徴

- ビジネスロジックを持たない
- Infrastructure に依存しない
- アプリケーション全体から参照可能

## テスト

`BuildInfo` は interface のため、テストでは mock を利用できます。

```bash
go:generate mockgen
```

例

```go
mockBuildInfo := mock_system.NewMockBuildInfo(ctrl)
mockBuildInfo.EXPECT().Version().Return("1.0.0")
```

## セキュリティ注意点

ビルド情報には次の情報を **含めないでください**

- 認証トークン
- 環境変数
- 秘密鍵
- 個人情報

含めるべき情報

- Version
- Git Revision
- Build Date

## まとめ

`internal/system` は

- ビルドメタ情報の提供
- バージョン管理
- 実行時メタデータ取得
- テスト可能な抽象化

を提供する **アプリケーションのランタイムメタ情報パッケージ**です。
