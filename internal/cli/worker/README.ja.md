# worker

[English](README.md) | 日本語

名前を指定した pull-ack worker を常駐プロセスとして起動し、終了シグナルを受け取るまで実行します。

## コマンド

```text
worker <worker-name> [args...]
```

## フラグ

このコマンドにフラグはありません。worker 名は必須で、後続の引数は worker の実装へそのまま渡されます。

## 使い方

```bash
# 名前を指定して worker を起動
./server worker myworker

# 追加の引数を worker へ渡す
./server worker myworker --some-arg value
```

## 注意点

- `job` と異なり、worker は常駐プロセスです。1 回の実行で終了せず、engine の完了チャネルを待ち続けます。
- SIGINT / SIGTERM を受けると engine を drain してグレースフルに停止し、実際の engine の終了結果（遅延した `Fatal` を含む）を必ず待ち切るため、失敗が握り潰されることはありません。
- engine 側が自走停止する場合（例: `Fatal` や unknown worker）もあり、その際はその結果でプロセスが終了します。
- グレースフルストップは 30 秒の停止タイムアウトで上限が設けられ、停止開始時点から計測されます。停止用 context はキャンセルを切り離しつつ trace/baggage は引き継ぎます。
- 本パッケージは、metrics/pprof サーバーとは別の専用 mux で health listener（`/healthz` liveness、`/readyz` readiness）を公開できます。
