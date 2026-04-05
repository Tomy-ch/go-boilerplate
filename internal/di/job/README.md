# job DI モジュール

概要:
このディレクトリは、ジョブ実行に関わる DI（依存性注入）コンポーネントを提供します。具体的には、登録されたジョブ群から `Runner` を組み立てるプロバイダー、ジョブ実行のための `State`（実行対象と完了チャネルの保持）、およびアプリケーション起動時に登録済みジョブを実行するライフサイクルフックを登録する仕組みを提供します。

## 役割

- `ProvideRunner`: `fx.In` で注入された `jobs` グループから `job.Runner` を生成して DI コンテナに提供します（`RunnerIn` 構造体）。
- `State`: 起動前に CLI 等で設定されたジョブ名・引数・完了通知チャネルを保持し、起動時フックで参照されます。スレッドセーフに `Set` / `Snapshot` を提供します。
- `RegisterJobHooks`: ライフサイクルのスタート時にジョブを非同期で実行するフックを登録します。ジョブが無ければシャットダウンをトリガーし、ジョブ実行後はシャットダウンを行います。

## 必要度

### 本番運用での必須度

- 必須度: 本番運用で推奨

- 理由: CLI 経由でのオンデマンドジョブ実行や、起動時に一度だけ実行するバッチ処理をサポートする場合に有用な DI 層です。ジョブ実行が運用要件に含まれるサービスでは導入が推奨されます。

### 開発/テスト運用での必須度

- 必須度: 開発/テスト運用で推奨

- 理由: 起動時ジョブや CLI ベースのジョブをテスト・デバッグする際に `State` を利用して実行対象を切り替えられるため、開発/テスト環境で便利です。

### 無効化した場合の影響

- DI モジュールを無効化すると、`Runner` の提供や起動時ジョブ実行フックが利用できなくなり、CLI 等からのジョブ実行が機能しなくなります。

## 注意点

- `RegisterJobHooks` は起動時に `State.Snapshot()` を参照して `done` チャネルが `nil` の場合は即座にシャットダウンを呼び出します。CLI などでジョブを実行する場合は、アプリケーション起動前に `State.Set(name, args, done)` を必ず呼んでください。
- ジョブ実行は別ゴルーチンで行われ、終了時に `done` チャネルへエラーを送信してからアプリケーションをシャットダウンします。
- `State` は内部で `sync.Mutex` を使っているため、並行アクセスは安全ですが、`done` チャネルのライフサイクル管理（誰がクローズするか等）は呼び出し側で設計してください。

## 実装の要点

- `RunnerIn` (fx.In): `Jobs []job.Job` を `group:"jobs"` として受け取り、`ProvideRunner` で `job.Runner` を生成します。
- `ProvideRunner`: `jobrunner.NewRunner(in.Jobs)` を呼んでランナーを返却します（`internal/controller/job` 側で `Runner` 実装を提供）。
- `RegisterJobHooks`: `lifecycle.Registrar` に Start フックを登録。スタート時に `State.Snapshot()` を取り、`done` が nil ならシャットダウン。そうでなければ `runner.Run` を呼び、結果を `done` に送ってから `Shutdowner.Shutdown()` を呼ぶ。
- `State`: `Set(name, args, done)` で事前セットし、フック側で `Snapshot()` により取得されます。

## 使い方

- 起動前に CLI 等で `State` にジョブをセットする流れの一例:

```go
done := make(chan error, 1)
state.Set("usercount", []string{"--verbose"}, done)
// アプリケーションを起動すると、Start フックがジョブを拾って実行し、完了後に done に結果を返す
err := <-done
if err != nil { /* ハンドリング */ }
```

- DI 組み込み例（fx のモジュールへ追加）:

```go
fx.Provide(
    job.ProvideRunner,
    job.NewState,
)
fx.Invoke(job.RegisterJobHooks)
```

## 前提 / 要件

- `fx` ライフサイクル（`fx.Shutdowner`）に依存してアプリケーションの終了をトリガーします。
- `job.Runner`（コントローラ層の実装）やログ/設定等の依存が DI 経由で提供されること。

## トラブルシューティング

- アプリ起動時に即座にシャットダウンされる: `State` に `done` チャネルがセットされていない可能性があります。CLI の初期化箇所で `State.Set` が呼ばれているか確認してください。
- ジョブが見つからずエラーになる: `ProvideRunner` に渡す `jobs` グループに目的のジョブが登録されているか確認してください。

## 拡張・カスタマイズ

- 起動時のジョブ実行を無効化したい場合は `RegisterJobHooks` の登録を行わないか、`State` のデフォルトを適切に設定してください。
- ジョブ実行前後のロギングやメトリクス収集を行いたい場合、`RegisterJobHooks` 内の goroutine 周りにラッパー処理を追加してください。
