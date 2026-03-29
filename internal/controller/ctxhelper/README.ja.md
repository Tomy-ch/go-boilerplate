# ctxhelper

[English](README.md) | 日本語

ctxhelperは「contextの利用を制御する境界レイヤ」です。

このパッケージは、context.ContextとEchoフレームワークのコンテキストを操作するためのヘルパー関数を提供します。

## 実装方法

このパッケージのコードは手動で実装せず、コード生成によって作成します。

生成機構の詳細については、以下を参照してください：

- `scripts/genctxkey/README.ja.md`

## 使用方法

ctxkeyを追加する場合は、以下のように `generate.go` に定義を追加します。

```go
//go:generate go run ../../../scripts/genctxkey --name UserID --type string --out .
```

その後、以下を実行します。

```bash
make gen-go-code
```

生成処理の詳細やオプションについては、[scripts/genctxkey/README.ja.md](../../../scripts/genctxkey/README.ja.md) を参照してください。

## 注意事項

### type の指定方法

ctxkey生成時の `--type` は以下のルールに従って指定します。

#### 基本型 / 同一パッケージ型

```bash
--type string
--type UserID
```

- import は不要です
- そのまま型として扱われます

#### 外部パッケージの型（推奨）

```bash
--type github.com/your/project/internal/domain/auth.Authn
```

- `<import-path>.<Type>` の形式で指定します
- generator が import と alias を自動解決します

生成されるコード例：

```go
import (
    auth "github.com/your/project/internal/domain/auth"
)

func GetAuthn(ctx context.Context) (auth.Authn, bool)
```

#### 注意

- `<import-path>` のみ（例: `github.com/foo/bar`）は指定できません
- 必ず型名まで含めて指定してください

### 編集について

本ディレクトリ内の `.gen.go` ファイルは自動生成されたコードです。

- 原則として手動編集は禁止です
- 変更は `scripts/genctxkey` を通して行ってください

例外として、依存解決のための軽微な修正（import調整など）は許可されますが、
恒久対応は generator 側で行うことを推奨します。
