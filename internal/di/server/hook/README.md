# server hook モジュール

概要: `internal/di/server/hook` は、アプリケーションサーバーのライフサイクルに結び付く各種フックを登録するためのモジュールです。HTTP サーバーの起動/停止フックおよびレートリミット用の定期クリーンアップフックを提供します。

## 役割

- `RegisterHTTPServerHooks`: Echo サーバー (`*echo.Echo`) の起動および停止用の Start/Stop フックを `lifecycle.Registrar` に登録します。起動時にサーバーを goroutine で開始し、ログを出力します。停止時には `e.Shutdown(ctx)` を呼んで安全に停止します。
- `RegisterRateLimitHooks`: IP ベースのレートリミッター (`ratelimit.IPRateLimiter`) 用のクリーンアップゴルーチンを Start/Stop フックとして登録します。設定で無効化されている場合は何も登録しません。

## 必要度

### 本番運用での必須度

- 必須度: 本番運用で必須

- 理由: HTTP サーバーの起動/停止および運用維持（ログや定期クリーンアップ）は本番環境で不可欠なため、このフック群は必須です。

### 開発/テスト運用での必須度

- 必須度: 開発/テスト運用で必須

- 理由: ローカル起動や自動テストでサーバーのライフサイクルを再現する必要があるため、フックの登録は重要です。テスト用にフックの差し替えや停止タイミングの制御ができることが望ましいです。

### 無効化した場合の影響

- HTTP サーバーを起動する Start フックやレートリミットの定期クリーンアップが登録されず、サーバー起動/停止や運用上のメンテナンス処理が正しく動作しなくなります。

## 注意点

- `RegisterHTTPServerHooks` は `extension.AppliedServerExtends` の注入を受けるため、サーバー拡張が適用された後に登録されることを前提にしています。
- HTTP サーバーの開始は goroutine 内で行われ、起動失敗はログに記載されますが、Start フック自体は非同期で終了するため、起動確認ロジックは別途必要な場合があります。
- `RegisterRateLimitHooks` は `ipCfg.Enabled()` を確認し、無効であれば登録を行いません。クリーンアップの起動は Start フック内で `time.Ticker` を使って行われ、Stop フックで安全に停止します。

## 実装の要点

- `newStartServerFunc`:
  - `srvCfg.Port()` からポートを取得し、`e.Start(":<port>")` を goroutine で実行します。起動ログにはポート/allowed_origins/cidr/mode 等を含めます。
- `newStopServerFunc`:
  - Stop フックで `e.Shutdown(ctx)` を呼び出し、停止ログを出力します。
- `rateLimitCleanupHook`:
  - Start: `time.NewTicker(ipCfg.CleanupInterval())` を作成し、ticker ごとに `rl.Cleanup()` を呼ぶループを別ゴルーチンで実行します。`startCtx.Done()` または内部の stop チャネルで終了します。
  - Stop: `stopCh` を `close` してループを終了させる。

## 使用例

DI 登録例（fx モジュール内）:

```go
fx.Invoke(
  hook.RegisterHTTPServerHooks,
  hook.RegisterRateLimitHooks,
)
```

HTTP サーバーの起動例 (ログ出力や設定は `config` 経由):

```go
// サーバーは Start フックで goroutine により起動されるため、アプリケーションを起動するだけで自動的に開始されます
```

## 前提 / 要件

- Echo (`*echo.Echo`) が初期化され、`RegisterHTTPServerHooks` に渡されていること。
- `ratelimit.IPRateLimiter` と `config.IPRateLimitConfig` が DI 経由で提供されていること（`RegisterRateLimitHooks` の前提）。

## トラブルシューティング

- サーバーが起動しない/すぐ停止する: ログに記載されたエラーを確認し、`extension` が正しく適用されているか、設定のポートが競合していないかを確認してください。
- レートリミットのクリーンアップが動作しない: `ipCfg.Enabled()` と `ipCfg.CleanupInterval()` の値を確認し、`rl.Cleanup()` を正しく実装しているか検証してください。

## モックとテストについて

- このパッケージには自動生成のモックは含まれていませんが、依存先（`ratelimit` や `logging` など）のモックを利用してフックの挙動をテストできます。
