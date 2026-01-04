# job hook

概要: `internal/di/job/hook` は、アプリケーション起動時に CLI 等で指定されたジョブを自動で実行するためのライフサイクルフックを登録するモジュールです。登録されたフックは起動時に `job.State` からジョブ名・引数・完了チャネルを取得し、非同期ゴルーチンで `job.Runner` を呼び出して実行結果を完了チャネルへ返送し、その後アプリケーションのシャットダウンをトリガーします。

## 役割

- `RegisterJobHooks` により、`lifecycle.Registrar` に Start フックを登録します。
- Start フックは起動時に `state.Snapshot()` を呼び、`done` チャネルが `nil` であればログ出力の後に `shutdowner.Shutdown()` を呼んで終了します。
- `done` が存在する場合は、別ゴルーチンで `runner.Run(startCtx, name, args)` を実行し、結果を `done` に送信した後 `shutdowner.Shutdown()` を呼びます。

## 必要度

### 本番運用での必須度

- 必須度: 本番運用で推奨

- 理由: バッチやワンショットで実行するジョブを CLI 経由で起動する運用がある場合、起動時にジョブを自動実行して安全にシャットダウンする仕組みは有用です。

### 開発/テスト運用での必須度

- 必須度: 開発/テスト運用で推奨

- 理由: 開発中にジョブ実行の自動化やデバッグを行う際に便利です。テストでは `job.State` をセットして起動時の振る舞いを確認できます。

### 無効化した場合の影響

- 起動時の自動ジョブ実行が行われなくなり、CLI からのワンショット実行フローが機能しなくなります。また、起動時にジョブが無いことを検出して即座にシャットダウンする既存の挙動が変わります。

## 注意点

- `state.Set(name, args, done)` をアプリケーション起動前に行っておく必要があります。`done` が `nil` の場合、フックはシャットダウンをトリガーします。
- Start フック内の実行は別ゴルーチンで行われるため、ジョブの実行は非同期に開始されます。完了は `done` チャネルで通知されます。
- `defer close(done)` によりフック側で `done` をクローズします。呼び出し側はクローズ後にチャネル受信を行う設計を想定してください。
- `shutdowner.Shutdown()` を呼んでアプリケーション停止をトリガーします。シャットダウンのタイミング設計に注意してください。

## 実装の要点

- 関数シグネチャ:
  - `RegisterJobHooks(reg lifecycle.Registrar, sd shutdowner.Shutdowner, runner job.Runner, logger logging.Logger, osCfg *config.OperationSystemConfig, state job.State)`
- Start フック内の処理:
  1. `name, args, done := state.Snapshot()` を取得
  2. `done == nil` の場合はログ出力後に `sd.Shutdown()` を実行して終了
  3. そうでなければ goroutine 内で `runner.Run(startCtx, name, args)` を実行し、結果を `done` に送信、`sd.Shutdown()` を実行

## 使用例

DI/FX での登録例:

```go
fx.Invoke(
  hook.RegisterJobHooks,
)
```

CLI 等からの起動フロー例（起動前に State をセット）:

```go
done := make(chan error, 1)
state.Set("usercount", []string{"--verbose"}, done)
// アプリケーションを起動すると Start フックでジョブが実行され、完了後に done へ結果が送られる
err := <-done
```

## 前提 / 要件

- `lifecycle.Registrar`、`shutdowner.Shutdowner`、`job.Runner`、`logging.Logger`、`job.State` が DI により提供されること。
- ジョブ実行は `runner.Run` によるため、ランナー側でジョブ名の検証や実行ロジックが実装されていること。

## トラブルシューティング

- アプリケーションがすぐシャットダウンする: `state.Snapshot()` の `done` が `nil` になっている可能性があります。CLI 側で `state.Set` を正しく呼んでいるか確認してください。
- ジョブが実行されない/不正なジョブ名エラー: `runner` に指定したジョブ名が登録されているか、`runner` の `Names()` を確認してください。

## モックについて

- モックディレクトリは自動生成の mock を想定しています。テスト時はモックを注入して `runner` や `shutdowner` の動作を制御できます。
