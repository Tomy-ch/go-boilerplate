---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [config]
---

# ADR-0038: サブシステムスコープの envPrefix 型付き設定ローダー

English canonical: [0038-subsystem-typed-config-loaders.md](../../adr/0038-subsystem-typed-config-loaders.md)

## ステータス

accepted

## 背景

アプリケーションは、OS 設定・アプリケーション識別情報・HTTP サーバータイムアウト・データベース接続・オブザーバビリティ・セキュリティ・認証・ワーカー・アウトボックスリレーなど、多くの関心事をカバーする環境変数から設定を読み取る。

すべての変数に対してフラットでスコープのない名前空間（例: `HOST`、`PORT`、`TIMEOUT`）を使用すると、変数の数が増えるにつれて衝突が発生し、変数名だけではどのコンポーネントに属するか判断できなくなる。すべての変数を内部グループなしの単一の構造体に解析すると、コードのナビゲーションが困難になり、各値のオーナーシップが不明確になる。

## 決定

各サブシステムは `internal/config/envspec.go` に独自の**型付き構造体**を持ち、ルートの `Loader` 構造体はすべてのサブシステムを対応する `envPrefix` タグとともに名前付きフィールドとして埋め込む。環境変数名は `{SUBSYSTEM}_{NAME}` の `UPPER_SNAKE_CASE` に従う。

`internal/config/envspec.go` のルート `Loader` は、各サブシステムを `envPrefix` タグ付きの名前付きフィールドとして埋め込む。構造のイメージ:

```go
// 省略形 — サブシステムの完全な一覧は internal/config/envspec.go を参照。
type Loader struct {
    OS     OperatingSystem `envPrefix:"OS_"`
    Server Server          `envPrefix:"SERVER_"`
    // … サブシステムごとに 1 フィールド
}
```

サブシステムの完全な一覧とフィールド詳細は `internal/config/envspec.go` と、ローディングフローの解説は `internal/config/README.md` を参照。

`env.ParseAs[Loader]()`（`github.com/caarlos0/env/v11` 経由）がプレフィックス付きの環境変数を自動的に対応する型付き構造体フィールドにマッピングする。

ロード後、`config.New()` は `Loader` を内部の `Config` 型に変換し、各コンポーネントが必要なフィールドのみを受け取れるよう、スコープを絞った SubConfig プロバイダー（`NewServerConfig`、`NewDatabaseConfig` など）を公開する。

## 影響

### ポジティブな影響

- プレフィックスから変数のオーナーシップが即座に明らかになる。`DB_HOST` は `Database` サブシステムに、`SERVER_PORT` は `Server` に属する。
- 新しいサブシステムの追加は限定的な変更で済む。`envspec.go` に構造体を追加し、`Loader` に `envPrefix` 付きフィールドを追加し、対応する SubConfig プロバイダーを追加するだけである。
- 各サブシステム構造体は独立して読み取り・テストできる。

### ネガティブな影響

- 新しいサブシステムの追加は `envspec.go`、`model.go`、`config.go`（Loader から Config への変換ステップ）にまたがる協調的な変更が必要になる。
- `envPrefix` の間接参照により、生の環境変数名と Go フィールド名が一致しない。オペレーターは `env/README.md` の正規テーブルを参照しなければならない。

## 検討した代替案

### すべての変数を単一のフラット構造体に収める

サブシステムのグループなしにすべての環境変数を 1 つの構造体に解析する。却下: 一般的な名前（例: `HOST`、`PORT`）での衝突には手動の曖昧さ解消が必要になり、大規模では構造体がナビゲート不能になる。

### サブシステムごとに 1 ファイル

サブシステムごとに個別の `envspec_*.go` ファイルを用意し、それぞれが独自の `ParseAs` 呼び出しを持つ。却下: 複数の解析パスにより、単一の検証済み `Loader` の利点が失われ、検証前の手動集約が必要になる。

## 補足

- 出典: `internal/config/envspec.go`（Loader 構造体の定義）、`env/README.md`（変数命名規則とサブシステムテーブル）。
- SubConfig プロバイダーパターンと不変性ルールは [ADR-0040](0040-immutable-fail-fast-config.ja.md) に記録されている。
- フィールドに `envDefault` と `required` のどちらを使用するかのガバナンスルールは [ADR-0039](0039-config-default-vs-required-governance.ja.md) に記録されている。
