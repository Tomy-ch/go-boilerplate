# job

[English](README.md) | 日本語

ジョブの定義・実行・状態管理のためのインターフェースを提供します。

## インターフェース

|インターフェース|説明|
|---|---|
|`Job`|`Name()` + `Execute(ctx, args)` — 単一ジョブの定義|
|`Runner`|`Run(ctx, jobName, args)` + `Names()` — 登録済みジョブの実行・一覧|
|`State`|`Set(name, args, done)` + `Snapshot()` — ライフサイクルフック用のジョブ実行状態保持|

## 設計意図

- 実行単位を抽象化し、実装依存を排除
- ジョブ定義（`Job`）とディスパッチ（`Runner`）とライフサイクル（`State`）を分離
- モック差し替えによるテスト可能なバッチ基盤

## 実装

- `Job`: `internal/controller/job/<name>/` にジョブごとに実装
- `Runner`: `internal/controller/job/` で登録済みジョブから組み立て
- `State`: `internal/controller/job/` でライフサイクルフック連携用に実装

## 注意点

- `Runner.Run` はジョブが見つからない場合、利用可能なジョブ名を含むエラーを返す
- 各ジョブは `context.Context` を尊重しキャンセル・タイムアウトに対応すること
- ジョブ名は一意である必要がある — 重複名は Runner 生成時にエラー
