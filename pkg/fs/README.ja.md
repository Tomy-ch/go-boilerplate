# fs

[English](README.md) | 日本語

ファイルシステム操作の薄いラッパーを提供し、利用側が `os` に直接依存せずインターフェース経由で扱えるようにします。

## ラップ対象

- `os.ReadFile` / `os.WriteFile`
- `path/filepath.Glob`

## 注意点

- `os` を使用するため、本パッケージは depguard の `reject_dangerous_os` を緩和しています（`!**/pkg/fs/**.go`）。
- domain / usecase 層からの import は禁止（depguard で強制）。ファイル I/O は外側の層の責務です。
- `FS` を注入してテストし、プロダクションは `OS{}` を結線します。生成モックは `mock/` 配下。
