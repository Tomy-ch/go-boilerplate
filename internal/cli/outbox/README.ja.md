# outbox-relay

[English](README.md) | 日本語

outbox relay プロセスを起動し、dead 行を復帰させる `replay` サブコマンドを提供します。

## 役割

このコマンドはトランザクショナル outbox パターンの「配信側」を実装するために存在します。メッセージは業務状態の変更と**同一のデータベーストランザクション**内で outbox テーブルに記録されるため、relay はその後にメッセージを publish するだけで二重書き込み問題（dual-write）を回避できます。コミット後の publish 失敗でメッセージが失われることも、publish は成功したがコミットされず幽霊メッセージが送られることもありません。常駐する relay は outbox を周期的に poll して pending 行を publish し、`replay` はリトライを使い切った行（`dead`）を `pending` に戻す運用上の復旧経路です。publish ループが独自のプロセスライフサイクルを持てるよう常駐型の独立したエントリポイントとし、その判定ロジックは薄いコマンド配線から切り離して単体テスト可能なコアに保っています。

## コマンド

```text
outbox-relay
outbox-relay replay [flags]
```

## フラグ

`outbox-relay` 本体にフラグはありません。

`outbox-relay replay`:

|フラグ|デフォルト|説明|
|---|---|---|
|`--message-id`|*(なし / 全 dead 行)*|対象の `message_id`。未指定の場合は全 `dead` 行が対象|

## 使い方

```bash
# relay を起動（SIGINT / SIGTERM まで常駐）
./server outbox-relay

# 全 dead 行を pending へ戻す
./server outbox-relay replay

# 特定の message_id の行のみ replay する
./server outbox-relay replay --message-id 1b4e28ba-2fa1-11d2-883f-0016d3cca427
```

## 注意点

- `outbox-relay` は outbox テーブルを周期的に poll し、未 publish のメッセージを送信して、終了シグナルを受け取るまで常駐します。
- シャットダウン時、停止用 context のタイムアウト（`ShutdownTimeout`）は停止開始時点から計測されるため、稼働時間に消費されません。
- `replay` は `dead` 行を `pending` へ戻し、再 publish の対象に復帰させます。
- `--message-id` は有効な UUID である必要があり、不正な値の場合は replay を実行する前にパースエラーを返します。
- `replay` は `pending` へ戻した行数を出力します。
