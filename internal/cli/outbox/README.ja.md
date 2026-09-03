# outbox-relay

outbox relay プロセスを起動し、dead 行を復帰させる `replay` サブコマンドを提供します。

## 役割

このコマンドはトランザクショナル outbox パターンの「配信側」を実装します。パターン自体は [`docs/design/outbox.md`](../../../docs/design/outbox.md) を参照してください。常駐する relay は outbox を周期的に poll して pending 行を publish し、`replay` はリトライを使い切った行（`dead`）を `pending` に戻す運用上の復旧経路です。publish ループが独自のプロセスライフサイクルを持てるよう常駐型の独立したエントリポイントとし、その判定ロジックは薄いコマンド配線から切り離して単体テスト可能なコアに保っています。

## コマンド

```text
outbox-relay --channel=<channel>
outbox-relay replay [flags]
```

## フラグ

`outbox-relay`:

|フラグ|デフォルト|説明|
|---|---|---|
|`--channel`|*(必須)*|このプロセスが担当する配送チャネル（`http` / `realtime`）。1 プロセスがちょうど 1 チャネルを捌き、既定値は持たない。誤ったチャネルで起動した relay は、本来のチャネルに担当者が居ない状態を作る。担当者の居ないチャネルは lag も記録されないため、滞留は「遅い」ではなく「見えない」形になる|

`outbox-relay replay`（チャネル非依存。dead 行は自分のチャネルを保持したまま戻る）:

|フラグ|デフォルト|説明|
|---|---|---|
|`--message-id`|*(なし / 全 dead 行)*|対象の `message_id`。未指定の場合は全 `dead` 行が対象|

## 使い方

```bash
# 1 つの配送チャネルを担当する relay を起動（SIGINT / SIGTERM まで常駐）
./server outbox-relay --channel=http

# 全 dead 行を pending へ戻す
./server outbox-relay replay

# 特定の message_id の行のみ replay する
./server outbox-relay replay --message-id 1b4e28ba-2fa1-11d2-883f-0016d3cca427
```

## 注意点

- `outbox-relay` は outbox テーブルを周期的に poll し、未 publish のメッセージを送信して、終了シグナルを受け取るまで常駐します。
- シャットダウン時、停止用 context のタイムアウト（`APP_SHUTDOWN_TIMEOUT`）は停止開始時点から計測されるため、稼働時間に消費されません。
- `replay` は `dead` 行を `pending` へ戻し、再 publish の対象に復帰させます。
- `--message-id` は有効な UUID である必要があり、不正な値の場合は replay を実行する前にパースエラーを返します。
- `replay` は `pending` へ戻した行数を出力します。
- CI はアプリケーションコードに触れるすべてのプルリクエストでこのエントリポイントを起動します（`.github/workflows/outbox-relay-boot-check.yaml`、[ADR-0091](../../../docs/adr/0091-ci-real-graph-boot-check.ja.md)）。`realtime` チャネルで relay を起動し、fx event logger からの `Application started` と `Application stopped` の両方を要求するため、グラフは組み立つが drain できない relay はこのチェックに失敗します。
