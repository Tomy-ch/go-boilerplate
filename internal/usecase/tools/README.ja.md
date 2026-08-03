# tools

[English](README.md) | 日本語

`internal/usecase/tools` は、**Usecase 層で再利用される小さなユーティリティ群**を格納するディレクトリです。

## サブディレクトリ

|パッケージ|説明|詳細|
|---|---|---|
|`paging/`|ページネーション（page/perPage → limit/offset 変換）と、top-N を含む共通の件数ポリシー|[README](paging/README.ja.md)|
|`search/`|検索キーワードのトークン化（分割、重複排除、上限制限）|[README](search/README.ja.md)|
|`money/`|マネー計算（最小単位整数・レート適用 half-up）|[README](money/README.ja.md)|

## 設計方針

- 複数の Usecase から共通利用されるユーティリティ
- ビジネスロジックを含まない — 機械的な変換のみ
- Infrastructure 依存なし
