# clifs

[English](README.md) | 日本語

CLI コマンド共通のファイルシステム操作ラッパー。利用側が `os` に直接依存せずインターフェース経由で扱えるようにし、オーケストレーションをユニットテスト可能にします。

## 公開 API

|型 / メソッド|説明|
|---|---|
|`FS`|ファイルシステム操作を抽象化するインターフェース|
|`OS`|`os` / `path/filepath` を用いた `FS` 実装|
|`FS.ReadFile(name) ([]byte, error)`|ファイル読み込み|
|`FS.WriteFile(name, data, perm) error`|ファイル書き込み|
|`FS.Glob(pattern) ([]string, error)`|glob マッチ|

## 注意点

- `os` の使用は depguard により `cmd` / `internal/cli` / `internal/config` / `scripts` のみ許可されるため、本パッケージは `internal/cli` 配下に置いています。
- `FS` を注入してテストし、プロダクションは `OS{}` を結線します。生成モックは `mock/` 配下。
