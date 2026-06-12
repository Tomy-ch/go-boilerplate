# exec

[English](README.md) | 日本語

外部プロセス実行の薄いラッパーを提供し、利用側が `os/exec` に直接依存せずインターフェース経由で扱えるようにします。

## 公開 API

|型 / メソッド|説明|
|---|---|
|`Runner`|コマンド実行を抽象化するインターフェース|
|`OS`|`os/exec` を用いた `Runner` 実装|
|`Runner.Output(ctx, dir, env, name, args) ([]byte, error)`|`dir` をカレントにコマンドを実行し標準出力を返す（標準エラーは `os.Stderr` へ）|

## ラップ対象

- `os/exec.CommandContext`

## 注意点

- `os` / `os/exec` を使用するため、本パッケージは depguard の `reject_dangerous_os` を緩和しています（`!**/pkg/exec/**.go`）。
- domain / usecase 層からの import は禁止（depguard で強制）。プロセス実行は外側の層の責務です。
- `Runner` を注入してテストし、プロダクションは `OS{}` を結線します。生成モックは `mock/` 配下。
