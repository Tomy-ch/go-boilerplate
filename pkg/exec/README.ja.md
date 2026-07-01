# exec

[English](README.md) | 日本語

外部プロセス実行の薄いラッパーを提供し、利用側が `os/exec` に直接依存せずインターフェース経由で扱えるようにします。

## ラップ対象

- `os/exec.CommandContext`

## 注意点

- `os` / `os/exec` を使用するため、本パッケージは depguard の `reject_dangerous_os` を緩和しています（`!**/pkg/exec/**.go`）。
- domain / usecase 層からの import は禁止（depguard で強制）。プロセス実行は外側の層の責務です。
- `Runner` を注入してテストし、プロダクションは `OS{}` を結線します。生成モックは `mock/` 配下。
