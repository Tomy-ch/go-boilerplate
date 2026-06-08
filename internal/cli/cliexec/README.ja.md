# cliexec

[English](README.md) | 日本語

CLI コマンド共通の外部プロセス実行ラッパー。利用側が `os/exec` に直接依存せずインターフェース経由で扱えるようにし、オーケストレーションをユニットテスト可能にします。

## 公開 API

|型 / メソッド|説明|
|---|---|
|`Runner`|コマンド実行を抽象化するインターフェース|
|`OS`|`os/exec` を用いた `Runner` 実装|
|`Runner.Output(ctx, dir, name, args) ([]byte, error)`|`dir` をカレントにコマンドを実行し標準出力を返す（標準エラーは `os.Stderr` へ）|

## 注意点

- `os` / `os/exec` の使用は depguard により `cmd` / `internal/cli` / `internal/config` / `scripts` のみ許可されるため、本パッケージは `internal/cli` 配下に置いています。
- `Runner` を注入してテストし、プロダクションは `OS{}` を結線します。生成モックは `mock/` 配下。
