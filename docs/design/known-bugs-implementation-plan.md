# 既知バグ 実装計画書（go-boilerplate release/v1.5.0）

- 入力: `/Users/tomy/dev/tmp/known-bugs.ja.md`（C1, C2, H1–H3, M1–M3 = 8件 open ＋ L1, L2 = 2件 new）
- 対象ブランチ: `release/v1.5.0` @ `edc83466`
- 方針の核: 横断パターン「**分類は揃っているが、それを消費する行動層が欠けている**」を埋める。参照実装は `internal/infrastructure/httpclient/`（classify→retry→backoff→breaker が完結）。
- 各修正は「負荷・競合・遅延・大入力でしか発火しない」性質のため、**発火条件を注入したテスト**を必ず添える（fake consumer の Extend 失敗 / fake DB の 40001 返却 / fake Sleeper / 並列更新など）。`make test` で 90% 維持。

> ## 改訂履歴（rev2: アーキ敵対レビュー反映）
> 4 クラスタの敵対的レビュー（各レイヤ README を SSOT として実読）で検出したアーキ的ズレを反映。主な変更:
> - **M3 を engine 行動層から撤回**（`controller/worker/README.md:31` の「再配送遅延=adapter 責務／Nack 遅延=port 非保証」に正面衝突。bug ではなく文書化済みの層分担）。
> - **C1 の seam を State 拡張 → hook の OnStop キャンセル方式へ変更**（worker hook の既存同型に揃える。boundary/mock 不変）。
> - **Phase 0 jitter を `pkg/backoff` → 新規 `pkg/jitter` へ**（backoff の純粋性宣言を保持）。
> - **H1 に tx.Manager 契約の明文化を必須追加**（fn が N 回再実行されうる＝非DB副作用の冪等性を boundary README + interface doc へ）。
> - **M1(b) を自前実装 → echo `middleware.ContextTimeout` ラップへ**（自前は再発明。不採用理由が事実誤認）。
> - **M1/M2 を Use → Pre middleware へ**（openapi validator(Use=6)の body 先読みで body limit が無効化する順序欠陥を回避）。
> - **C2 に第3タイムアウト（cli ハードコード 30s）と config 軸重複（既存 `APP_SHUTDOWN_TIMEOUT`）を追記**。

---

## 0. 先行リファクタ（共有プリミティブ）

H1（tx retry）/ 既存 httpclient が同じ「指数 backoff ＋ full jitter」を必要とする。現在 jitter は `httpclient/retry.go:fullJitter` に閉じている。重複を避けるため共有化する。

> **【rev2 アーキ修正】** 当初案の `pkg/backoff/jitter.go` は **不可**。`pkg/backoff/backoff.go:1-2` の **package doc** が「現在時刻や乱数には依存しません」を**パッケージ単位の契約**として宣言しており、別ファイルでも `math/rand/v2` を入れた時点で契約が偽になる。既存設計も `httpclient/retry.go:67` で `fullJitter(exp.Duration(...))` と純粋 backoff の**外側**で jitter を適用しており、純粋性維持は設計意図そのもの。

- **`pkg/jitter/`（新規パッケージ）**: `func Full(d time.Duration) time.Duration`（`math/rand/v2` 依存を package doc に明記）。実装は既存 `fullJitter` を移設。
- **`pkg/jitter/README.md`（新規）** を追加し、`pkg/README.md` のパッケージ一覧表にも 1 行追加（`pkg/README.md` の「Checklist for Adding a New Package」要件）。
- `httpclient/retry.go:114` の `fullJitter` を `jitter.Full` 呼び出しに置換（挙動不変）。
- pkg ルール: `pkg` は `internal` 非依存・framework 非依存・単一責務。決定論計算（`backoff.Exponential.Duration`）と乱数適用（`jitter.Full`）を別パッケージに分離することで両方満たす。

> この節を H1 より先に入れると、H1 がそのまま `jitter.Full` を使える。M3 は撤回したため jitter 利用者ではなくなった。

---

## Phase 1: 局所修正（新規 config 不要・低リスク）

### H2. heartbeat の Extend 失敗を可視化する

- **場所**: `internal/controller/worker/run.go:226`（`startHeartbeat` 内）
- **変更**:
  ```go
  case <-ticker.C:
      if err := r.consumer.Extend(ctx, m, interval*extendVisibilityFactor); err != nil {
          if ctx.Err() != nil {
              return // 停止中の Extend 失敗は握り潰してよい
          }
          r.e.met.ExtendError(ctx)
          r.e.log.Named("worker.extend").Warn(
              "extend failed",
              append(msgFields(ctx, r.name, m), logging.Error(logging.ErrorKey, err))...,
          )
      }
  ```
- **付随**: `observability.WorkerMetrics` に `ExtendError(ctx)` カウンタを追加。既存カウンタ（`Retried` / `PollError` 等）と同じ `meterBuilder` 定義方式・**ドット区切り meter 名**（例 `worker.extend_error`。既存 `worker.received` 等の命名規約に合わせる。`_total` サフィックスは付けない）に倣う。
- **テスト**: `consumer.Extend` が error を返す fake で、warn ログ・metric 計上を検証。`ExtendInterval` を極小にして tick を確実に走らせる。最低でも「握り潰さない」を固定する。

### H3. job 停止フェーズの失敗を exit code に反映する

- **場所**: `internal/cli/job/job.go:62-66`（`gracefulStop`）＋ `runJob` の 2 つの return 分岐
- **変更**: `gracefulStop` を error 返却に変え、本体結果と `errors.Join` する。
  ```go
  func gracefulStop(ctx context.Context, stop StopFunc) error {
      stopCtx, cancel := context.WithTimeout(ctx, stopTimeout)
      defer cancel()
      return stop(stopCtx)
  }
  // runJob:
  case err := <-done:
      return errors.Join(err, gracefulStop(ctx, stop))     // timeout<=0 経路も同様
  case <-waitCtx.Done():
      return errors.Join(waitCtx.Err(), gracefulStop(ctx, stop))
  ```
- `errors.Join` は nil を畳むので、停止成功時は本体 err のみ。`os.Exit(1)` 判定（Execute 側）が停止失敗（OTel flush 失敗等）も拾う。
- **テスト**: `stop` が error を返す fake で、(a) 本体成功＋停止失敗→非 nil、(b) 両方成功→nil、(c) timeout 分岐＋停止失敗→両方が `errors.Is` で取れる、を検証。

### L2. HTTP client `Do()` 入口の precondition 検証

- **場所**: `internal/infrastructure/httpclient/client.go:64-66`（既存 `AllowRetry` チェックの直後）＋ `retry.go`
- **変更**: known-bugs.ja.md の修正方針コードをそのまま採用。
  ```go
  if !isKnownMethod(req.Method) {
      return nil, xerrors.Wrap(apperror.ErrInvalidArgument, "unknown HTTP method")
  }
  if req.Downstream == "" {
      return nil, xerrors.Wrap(apperror.ErrInvalidArgument, "Downstream is required")
  }
  ```
  `isKnownMethod` は `MethodGet..MethodDelete` の 5 定数 membership 判定（`retry.go` に追加、`isRetrySafe` の switch と定義を共有してもよい）。
- **テスト**: `Method:""` / `Method:"Get"`（大小違い）/ `Downstream:""` がいずれも送信前に `ErrInvalidArgument` で弾かれること。正規定数は従来どおり通ること（既存テストの回帰確認）。

### L1. SSRF dial guard に CGNAT 帯を追加

- **場所**: `internal/observability/http_client_transport.go:103-119`（`guardedDialControl`）
- **変更**: deny 判定に 100.64.0.0/10（RFC 6598）を追加。`net/netip` で package 変数として一度だけ parse。
  ```go
  var cgnatPrefix = netip.MustParsePrefix("100.64.0.0/10")

  // private/loopback 不許可時のブロック条件に追加:
  if !allowPrivateNetworkFromContext(ctx) &&
      (ip.IsLoopback() || ip.IsPrivate() || isCGNAT(ip)) {
      return fmt.Errorf("ssrf guard: blocked private/loopback address %s", host)
  }
  ```
  `isCGNAT` は `netip.Addr` 変換して `cgnatPrefix.Contains`。`net.IP`↔`netip` 変換に注意（`netip.AddrFromSlice`）。
- **任意**: 網羅性を上げるなら 192.0.0.0/24・198.18.0.0/15・240.0.0.0/4 も同 list 化。脅威モデル依存なので CGNAT を必須・他は任意とする。ULA(fc00::/7) は `IsPrivate()` 済みで追加不要。
- **テスト**: `100.64.x.x` が allow フラグ無しで block、allow フラグ有りで通過。`169.254.169.254`（メタデータ）は従来どおり常時 block の回帰確認。

---

## Phase 2: worker の行動層を埋める

### M3. retryable Nack の backoff 〔engine 行動層としては撤回〕

> **【rev2 アーキ判定: CONFIRMED 違反 → engine 実装を撤回】**
> `internal/controller/worker/README.md:31` が明文で設計判断済み:
> 「per-message redelivery delay is the adapter's best-effort `Nack` behavior (e.g. SQS visibility) … the `Nack` delay is **not** a port guarantee; broker-agnostic backpressure is the circuit's job.」
> 当初の M3 案（engine 内で `Consumer.Extend` を backoff 転用＋Ack しない）は次の 3 点で文書化済みアーキに違反する:
> 1. 「per-message 再配送遅延 = adapter / IaC の責務」という README の層分担を engine 側から反転。
> 2. `Extend`（`consumer.go:18` =「**処理中** lease 延長＝ハートビート」）を、handler 失敗後（`finishMessage` で in-flight 解放済み＝処理は終了）の再配送スケジューリングへ転用 — 動詞の意味論を破壊。
> 3. `Settings`（`settings.go:18`）と `envspec.go:20` が二重に謳う「**broker 非依存**」を、visibility という broker 固有 primitive 依存の概念で汚染。circuit breaker は Receive をゲートするだけで broker primitive 不使用＝真の非依存であり、NackBackoff は同列に正当化できない。
>
> CLAUDE.md「Do NOT introduce new architectural patterns without instruction」に抵触するため、**engine への NackBackoff 実装・`Settings`/`WORKER_` config 追加は行わない**。

**改訂方針（README の意図を維持）**:
- 個別メッセージの再配送 backoff は **adapter（SQS visibility timeout / redrive policy）＋ IaC の責務**とし、boilerplate engine は既存の circuit breaker で intake backpressure を担う現状の分担を維持する。これは known-bugs.ja.md の「対象外: per-message backoff の閾値は利用者文脈」とも整合する（バグではなく設計判断）。
- known-bugs.ja.md の「boundary README に明記が要るか」は**誤り**——意図は `controller/worker/README.md` に既出で、M3 案とは**逆**。むしろ「engine は per-message delay を持たない」を再確認する一文を README に残すか検討。
- **どうしても** engine seam が要る場合のみ（bug-fix の範囲外・別 PR・要ユーザー判断）: `Extend` 転用ではなく `Consumer` port に明示的な `NackWithDelay(ctx, m, d)` を追加し、「delay 非保証」契約を破らず adapter 実装へ委ねる。これは README の "different layers" 宣言の改訂を伴う**設計変更**であり、本書のバグ修正スコープから切り離す。

→ M3 は「未配線バグ」ではなく「文書化済みの層分担」。**本計画から除外**し、production-checklist 側（adapter/IaC の関心）へ移送する。

### C2. worker drain とプロセス停止の順序を保証する

- **場所**: `internal/di/worker/hook/register_worker_hooks.go:48-56`（OnStop）/ `internal/di/worker.go`（fx.App 構築）/ `internal/cli/worker/worker.go`（停止 ctx）/ config
- **根本（診断は CONFIRMED）**: `DrainTimeout`(engine, 既定 30s) が fx の StopTimeout（明示設定なし＝**既定 15s**）より長いと drain 途中で hook を抜け、後続の DB pool close と in-flight handler が競合。**現状の既定値で既に発火する**。

> **【rev2 アーキ修正: 当初案は INCOMPLETE】**
> **第3のタイムアウトを見落としていた**。実効カットオフ = `min(app.Stop に渡す ctx の deadline, fx StopTimeout)`。
> - `internal/cli/worker/worker.go:9` に **ハードコード `const stopTimeout = 30 * time.Second`** があり、`gracefulStop`（同 :60-63）が `app.Stop` へ渡す ctx の deadline = now+30s。
> - つまり drain 完了には「fx StopTimeout > DrainTimeout」**かつ**「cli stopTimeout > DrainTimeout」の両方が必要。fx.StopTimeout だけ伸ばしても cli の 30s 定数が binding なら drain は 30s 境界で競合し続ける（DrainTimeout 既定 30s とデッドヒート）。
> - **config 軸の重複**: 既存 `APP_SHUTDOWN_TIMEOUT`（`Application.ShutdownTimeout`, `envspec.go:47`）が serve 経路（`cmd/serve.go:55`）で shutdown grace として使われている。worker 経路だけが cli の 30s 定数を使い `APP_SHUTDOWN_TIMEOUT` を無視。ここに新 `WORKER_SHUTDOWN_GRACE` を増やすのは既存 shutdown-grace 軸の二重化（memory: 「軸を混ぜない／根拠を言えない config 値を置かない」に抵触）。

- **変更（改訂）**:
  1. **grace の出所を一本化**: まず既存 `APP_SHUTDOWN_TIMEOUT`（`Application.ShutdownTimeout`）を worker 経路の grace としても流用できないか検討する。worker 固有 grace が必要だと根拠を言える場合のみ `WORKER_SHUTDOWN_GRACE` を新設。**新 config はデフォルト方針として追加しない**。
  2. **3 つのタイムアウトを整合**: `internal/di/worker.go` の `fx.New(...)` に `fx.StopTimeout(grace)` を追加し、**かつ** `cli/worker/worker.go:9` のハードコード `stopTimeout` を grace と統一（定数のままだと validation が嘘になる）。
  3. **起動時 validation（fail fast）を grace 適用箇所に co-locate**: `DrainTimeout < min(grace, cli stopTimeout)` を、**grace を実際に適用する `internal/di/worker.go`**（`fx.StopTimeout` を書く場所）で検証し違反時 error。
     > 当初案の「`ProvideEngine`(`di/worker/runner.go`, package `worker`)に置く」は**precedent 不整合**。参照すべき `extension.validatePriorityConflicts`（`extension.go:142`）は middleware を**実際に適用する** `applyMiddlewares`（同:116, 同一パッケージ）と co-located。検証と enforcement の同居が precedent。grace を適用する `di/worker.go` と検証する `ProvideEngine` を別パッケージに分離するとドリフトで validation が無意味化する。
  4. **OnStop は drain 完了を基本に待つ**: `select { engineDone / stopCtx.Done() }` は保険として維持。validation で `DrainTimeout < min(grace, cli stopTimeout)` が保証されれば通常 `engineDone` が先勝ち。猶予超過時の挙動（未 Ack＝再配送）を doc に明記。
- **テスト**: (a) `DrainTimeout >= min(grace, cli stopTimeout)` で起動 validation が error、(b) 正常値で成功。drain の実時間競合は単体困難なため validation のロジックテストで担保。

---

## Phase 3: tx の行動層を埋める

### H1. tx manager に serialization failure / deadlock のリトライを入れる

- **場所**: `internal/infrastructure/rdb/driver/transaction.go`（`Do`）/ `pgerror/pgerror.go`（分類）/ config / `di/module`（NewTransactionManager 結線）
- **分類層の整備**: `pgerror` に retryable tx 専用判定を追加（既存 `IsLockNotAvailable` と同形）。
  ```go
  // 40001 = serialization_failure, 40P01 = deadlock_detected
  func IsRetryableTxError(err error) bool {
      var pgErr *pgconn.PgError
      return errors.As(err, &pgErr) && (pgErr.Code == "40001" || pgErr.Code == "40P01")
  }
  ```
  ※ `sqlstateToAppError` の `ErrUnavailable` 写像は維持（最終的に諦めたときの返り値はそのまま）。retry 判定は写像後の sentinel ではなく**生 SQLSTATE**で行うのが安全（接続断 ErrUnavailable まで巻き込まない）。
- **行動層**: `Do` を有限リトライでラップ。nested tx（ctx に既存 tx）経路は**リトライ対象外**（savepoint 相当・1 回のみ）。
  ```go
  func (t *txManager) Do(ctx context.Context, fn func(context.Context) error) error {
      if _, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
          return fn(ctx) // ネストは即実行（retry しない）
      }
      var err error
      for attempt := 1; attempt <= t.maxAttempts; attempt++ {
          err = t.doOnce(ctx, fn) // 現行の begin→fn→commit を doOnce へ抽出
          if err == nil || !pgerror.IsRetryableTxError(err) || attempt == t.maxAttempts {
              return err
          }
          wait := jitter.Full(t.backoff.Duration(attempt - 1)) // Phase 0 の新 pkg
          if serr := t.sleeper.Sleep(ctx, wait); serr != nil {
              return err // ctx 打ち切りは元エラーを返す
          }
      }
      return err
  }
  ```
  `doOnce` は現行 `Do` の本体（begin / defer rollback / commit）をそのまま移設。`wait` は `jitter.Full(t.backoff.Duration(attempt-1))`（Phase 0 の新 pkg）。
- **依存追加**: `txManager` に `sleeper clock.Sleeper` / `maxAttempts int` / `backoff backoff.Exponential` を持たせ、`NewTransactionManager(db, logger, sleeper, cfg)` へ拡張。`sleeper` は `system.NewSleeper()`（既存実装）を DI 結線。infra→`usecase/boundary/clock` 依存は `infrastructure/README.md:41-43`（`Infra --> Usecase` 許可）＋ httpclient の先例（`client.go:39` が既に `clock.Sleeper` 注入）と整合＝OK。
- **config 追加**: `DB_TX_MAX_RETRIES` / `DB_TX_RETRY_BASE_BACKOFF` / `DB_TX_RETRY_MAX_BACKOFF`（isolation level・上限は利用者文脈）。

> **【rev2 アーキ必須: tx.Manager 境界契約の明文化】**
> 配置（Manager 実装でのリトライ）は妥当（begin/commit の唯一の所有者・httpclient substrate と同型）。**ただしリトライ導入は `fn` が最大 N 回再実行されうるという境界契約の意味変更**。`fn` は usecase のビジネスロジックで DB 外副作用（イベント発行・outbox 書込・メール送信・メモリ変更）を含みうるため、serialization リトライ時に**多重発火**しうる（C/H 級と同じ不可視性）。「fn の冪等性前提」を**コードコメントでなく SSOT に固定**する（README > Code）:
> 1. `internal/usecase/boundary/tx/README.md` の Notes に明記:「`Do` は serialization failure / deadlock 検出時に `fn` を最大 N 回再実行しうる。よって `fn` は DB 副作用以外について冪等であること（呼出側責務）。nested(既存 tx 再利用)経路はリトライ対象外＝1 回」。
> 2. `tx_manager.go:11` の `Manager` interface doc にも同趣旨を 1 行追加。
> 3. **既存ドリフトの同時修正**: `internal/infrastructure/rdb/driver/README.md:223` は旧シグネチャ `NewTransactionManager(cfg, db, logger)` を記載（実コード `transaction.go:26` は `(db, logger)`）。今回 `sleeper`/`cfg` を足すので README も実体に正す。
> 4. **付随ドリフト**: `usecase/boundary/clock/README.md` は `Clock`/`Now()` のみ記載で `Sleeper`（`clock.go:18`）が未文書。H1 が 2 人目の消費者になるので README に `Sleeper` を追記。
- **テスト**: fake DatabaseDriver で (a) 40001 を N-1 回返し N 回目 commit 成功→最終 nil・試行回数検証、(b) 40P01 も同様、(c) 23505（非 retryable）は即 return・1 回のみ、(d) maxAttempts 到達で最後の err を返す、(e) ネスト tx は retry しない、(f) Sleep が ctx.Err→打ち切り。fake Sleeper で待ち時間は即時化。

---

## Phase 4: 境界の上限・期限（REST / DB）

### M1. クエリ実行時間の上限（statement_timeout ＋ per-request deadline）

- **(a) DB 側**: `internal/infrastructure/rdb/driver/driver.go:newDB`。`pgxpool.ParseConfig` 後に接続 RuntimeParams を設定。
  ```go
  if v := dbConnCfg.StatementTimeout(); v > 0 {
      poolCfg.ConnConfig.RuntimeParams["statement_timeout"] = strconv.Itoa(int(v.Milliseconds()))
  }
  if v := dbConnCfg.LockTimeout(); v > 0 {
      poolCfg.ConnConfig.RuntimeParams["lock_timeout"] = strconv.Itoa(int(v.Milliseconds()))
  }
  ```
  config 追加: `DB_STATEMENT_TIMEOUT` / `DB_LOCK_TIMEOUT`（`DBConnectionConfig` に getter 追加。安全側デフォルトを置き値は利用者上書き）。`55P03`(lock_timeout) は `pgerror` で既に `IsLockNotAvailable` あり、`57014`(statement_timeout) は `ErrUnavailable` 写像済み。
- **(b) REST 側**: per-request timeout middleware を新設。

> **【rev2 アーキ修正: 自前実装は再発明（REFUTED）】**
> echo 標準に **`middleware.ContextTimeout` / `ContextTimeoutWithConfig`** が既存で、計画の擬似コードと**同一実装**（`context_timeout.go:97-100`: `WithTimeout(c.Request().Context())` → `c.SetRequest(... WithContext)` → `defer cancel()`）。当初の不採用理由「echo `middleware.Timeout` は handler ctx をキャンセルしない」は**事実誤認**——`Timeout` の真の問題は response writer のデータ競合で、echo 自身が `Deprecated` 注記で「代わりに race-free な `ContextTimeout` を使え」と公式案内（`timeout.go:63-67`）。自前実装は `defer cancel()` 漏れ risk も抱える。

  - `internal/controller/httpstack/timeout/timeout.go`（新規。パッケージ名は単語小文字規約に合わせ `httptimeout` ではなく `timeout`）: `middleware.ContextTimeoutWithConfig` を薄くラップ（`cors.Middleware` が `middleware.CORSWithConfig` をラップする `cors.go:15-17` と同型）。公開関数は規約どおり `Middleware`。
  - **error 統合（当初案に欠落）**: `ContextTimeout` 既定 ErrorHandler は `echo.NewHTTPError(503)` を返す（`context_timeout.go:82-88`）が、本 repo は統一 errorhandler（`outbound.ErrorHandlerModule()`, apperror→status 写像）を持つ。`ContextTimeoutConfig.ErrorHandler` で deadline 超過を `apperror`（`ErrUnavailable` 等）に wrap し統一 errorhandler に委譲、body 形を揃える。
  - **配置: Use ではなく Pre**（下記 M2 と共通の理由）。`internal/di/server/extension/inbound/timeout_di.go`（新規）が `extension.PreMiddlewareOut` を返し、**timeout=Pre priority 3**。deadline ctx は `c.SetRequest` で request ctx を差替えるため、後続の全 Use（observability/logging 等は `c.Request().Context()` から派生）＋ openapi 検証 ＋ handler ＋ DB を一括で覆える。`server.go:MiddlewareModule()` に module 追加。
  - config 追加: `REST_REQUEST_TIMEOUT`。
- **テスト**: driver は RuntimeParams 設定の有無を検証（poolCfg レベル）。middleware は `httptest` で「deadline 超過時に handler ctx が Done＋統一 errorhandler 経由の body」「正常時は素通り」を検証。

### M2. リクエストボディのサイズ上限

> **【rev2 アーキ修正: Use 採番は致命的順序欠陥（CONFIRMED・本クラスタ最危険）】**
> openapi validator（`openapi_di.go:15` で **Use priority=6**）は requestBody 宣言のある operation で**ボディを decode/buffer する**（kin-openapi）。Use の空きは **9 と 11+ のみ**で 1〜6 の間に整数の空きが無いため、当初案「Use・衝突回避で採番」だと body limit が priority 9 となり、**openapi(6) が先に無制限ボディを読み切る → body limit が validated な POST/PUT で実質無効化**。「メモリ圧迫を防ぐ安全機構が、最も攻撃面である body 付き endpoint で効かないのに『実装済み』に見える」サイレント欠陥。

- **場所**: **Pre middleware**（ルーティング前＝全 Use・openapi より確実に前。echo `BodyLimit` は reader をラップするだけでルーティング非依存）。
  - `internal/controller/httpstack/bodylimit/body_limit.go`（新規）: `middleware.BodyLimit(limit)` を薄くラップ（cors 先例と同型）。
  - `internal/di/server/extension/inbound/body_limit_di.go`（新規）: `extension.PreMiddlewareOut`、**bodyLimit=Pre priority 2**（Pre 名前空間は uri=1 のみ＝無衝突）。`MiddlewareModule()` に追加。
  - config 追加: `REST_BODY_LIMIT`（例 `"1M"`。echo は文字列形式を受ける。上限超過は `echo.ErrStatusRequestEntityTooLarge`=413）。
- **補足**: outbound 側は `httpclient.readBody` の `MaxResponseBytes` で既に上限化済み。穴は inbound のみ。
- **配置メモ**: timeout/body limit を `inbound/` に置くか `security/`（DoS ガード解釈）に置くかは解釈余地あり。`extension/inbound/README.md` の定義（入口の品質/安全/一貫性）に概ね整合するため `inbound/` を採るが、「なぜ security でなく inbound か」を README に一言残す（extension README が priority 管理表の維持を要求）。
- **テスト**: `httptest` で上限超過 body が 413、上限内は通過。openapi 検証付き route でも上限が効くことを 1 ケース固定（Pre 配置の回帰防止）。

#### Pre/Use priority 一覧（実コード抽出）と新規採番

Pre と Use は **別 priority 名前空間**（kind 単位で `validatePriorityConflicts`）。

| 種別 | 既存（name=value） | 新規（rev2 推奨） |
|---|---|---|
| **Pre** | uri=1 | **bodyLimit=2**, **timeout=3** |
| **Use** | requestID=1, observability=2, recovery=3, cors=4, security=5, openapi=6, forceJSON=7, logging=8, cookie=10 | （追加しない） |

両方を Pre に寄せることで「body は validator が読む前に上限化／deadline は検証・handler・DB を覆う」が同時成立し、混雑した Use 帯（既存 9 定数）の renumber を一切伴わずに導入できる。

---

## Phase 5: job 実行タイムアウトの配線（Critical）

### C1. `--timeout` を実行中断として機能させる

- **根本**: timeout（CLI flag）は `runJob` の `waitCtx`（待ち時間）にしか効かず、実行 ctx に伝播しない。実行 ctx は hook で `context.WithoutCancel(startCtx)` 生成され、**cancellation も deadline も持たない**。fx OnStart の `startCtx` は `app.Start(ctx)` に StartTimeout を被せ OnStart 完了で cancel する別 ctx で、かつ `WithoutCancel` が deadline ごと落とすため、**ctx 経由で deadline を素通しできない**（計画の主張は正しい）。

> **【rev2 アーキ修正: seam を State 拡張 → hook の OnStop キャンセルへ変更】**
> 当初の「timeout を `job.State` に載せて hook へ運ぶ」案は**動くが最善でない（PLAUSIBLE）**。`usecase/boundary/job/README.md:13` の State 責務は「invocation identity(name/args)＋結果 channel」で、timeout（実行ポリシー／orchestration 関心）を boundary に押し込むと usecase boundary に orchestration が滲む。さらに当初案は (a) state 構造体の場所を誤記（boundary ではなく **`internal/controller/job/state.go:9`**）、(b) 波及を過少計上（`controller/job/state.go` + `state_test.go` + boundary README en/ja + mock 再生成まで実波及）。
>
> **決定的事実**: `ctx 経由で deadline を運べない`は「データ(time.Duration)で運べ」を意味せず「**キャンセルで中断せよ**」を排除しない。そして worker hook が**まさにその最小 seam を実装済み**（`register_worker_hooks.go`: `context.WithCancel(context.Background())` + `reg.RegisterStop(cancel)`）。job hook はこの兄弟パターンに対し**唯一 `WithoutCancel(startCtx)` を使い RegisterStop を持たない outlier**。C1 は本質的に「job hook が worker hook と同じ OnStop キャンセル配線を忘れている」だけ。CLAUDE.md「新パターン導入禁止」にも、既存パターン踏襲＝整合。

- **改訂 seam（boundary 不変・hook 1 パッケージで完結）**:
  1. `internal/di/job/hook/register_job_hooks.go`: 実行 ctx をキャンセル可能化し、OnStop で cancel を登録。
     ```go
     // RegisterJobHooks 内（worker hook と同型）
     jobCtx, cancel := context.WithCancel(context.WithoutCancel(startCtx)) // 起動キャンセル分離は維持しつつ、停止で切れる
     reg.RegisterStart(func(startCtx context.Context) error {
         go runJobAndShutdown(jobCtx, ...) // runner.Run(jobCtx, ...) へ
         return nil
     })
     reg.RegisterStop(func(_ context.Context) error { cancel(); return nil })
     ```
  2. timeout の発火経路は**既存のまま**：`runJob`（cli, timeout 既知）→ `waitCtx` 失効 → `gracefulStop` → `stop`(=`app.Stop`) → fx が OnStop 実行 → `cancel()` → `runner.Run` の ctx キャンセル → pgx が ctx 尊重で DB クエリ中断。`runJob` は全分岐で必ず `gracefulStop` を呼ぶ（`cli/job/job.go:42,51,56`）ので確実に走る。
  3. **触らない**: `usecase/boundary/job/job.go`（interface）、mock、`controller/job/state.go`、`cli`・`di` の `StartFunc` 型、`cmd/job.go`、boundary README。State 案で波及した全箇所を回避。
- **契約整合**: `internal/cli/job/README.md:32` が既に「`--timeout` set → the job is **cancelled** if it exceeds the specified duration」と**キャンセルで規定**。本 seam はこの語彙に忠実（deadline データ搬送より整合）。
- **C2 との同型化**: 本 seam により C1（OnStop キャンセル）と C2（OnStop drain 順序）が「detached goroutine の OnStop teardown」という同一問題系に揃い、worker hook が両者を一体で解く事実とも符合。
- **実装時の検証点**: job goroutine は完了時に自ら `shutdown(sd)`（`sd.Shutdown()`）を呼ぶ（worker には無い差分）。timeout 時は cli が `app.Stop` を呼びつつ、cancel された runner.Run 完了で goroutine も `sd.Shutdown()` を呼ぶため、**app.Stop と Shutdown の二重停止が競合しないか**を実装・テストで確認（fx Shutdowner の冪等性／OnStop 内 cancel→goroutine 完了の順序）。
- **テスト**: fake runner（ctx.Done を観測してから返る）で (a) timeout 経由の `app.Stop`→OnStop→runner ctx が Done、(b) startCtx を後からキャンセルしても jobCtx は巻き込まれない（`WithoutCancel` の意図維持）、(c) 正常完了時は OnStop cancel が副作用を持たない。

---

## config 追加の一覧（`new-env` skill 推奨）

下記はいずれも `internal/config/{envspec,model,config}.go` ＋ mock ＋ `env/.env.*` ＋ `env/README.{md,ja.md}` を横断更新する。手作業ではなく **`new-env` skill を 1 変数ずつ**回すのが安全。

| 変数 | subsystem | 用途 | 既定の考え方 |
|---|---|---|---|
| `DB_TX_MAX_RETRIES` | rdb | H1: tx リトライ上限 | 例 3 |
| `DB_TX_RETRY_BASE_BACKOFF` | rdb | H1: tx リトライ初期 backoff | 数 ms |
| `DB_TX_RETRY_MAX_BACKOFF` | rdb | H1: tx リトライ上限 backoff | 数十〜百 ms |
| `DB_STATEMENT_TIMEOUT` | rdb | M1: statement_timeout | 安全側 default |
| `DB_LOCK_TIMEOUT` | rdb | M1: lock_timeout | 安全側 default |
| `REST_REQUEST_TIMEOUT` | rest | M1: per-request deadline | 安全側 default |
| `REST_BODY_LIMIT` | rest | M2: body 上限 | 例 `1M` |

> **rev2 での増減**:
> - 削除: `WORKER_NACK_BACKOFF_BASE/MAX`（M3 撤回）。
> - 保留（新設しない方針）: `WORKER_SHUTDOWN_GRACE`。C2 はまず既存 `APP_SHUTDOWN_TIMEOUT` 流用を検討し、worker 固有 grace の根拠が立つ場合のみ新設。
> - 注: subsystem prefix（`DB_` / `REST_` 等）は `envspec.go` 実体に合わせて要確認。`REST_` prefix の有無は実体未確認のため新規追加時に `new-env` が検出する subsystem に従う。

---

## 推奨実装順序（PR 分割）

1. **PR0 先行リファクタ**: 新規 `pkg/jitter`（`Full` 抽出 ＋ README ＋ `pkg/README.md` 表追記）＋ httpclient 置換。挙動不変。
2. **PR1 局所修正**: H2 / H3 / L1 / L2（新 config なし・低リスク・レビュー容易）。H2 の metric 追加のみ observability に触れる。
3. **PR2 worker stop 整合**: C2（fx.StopTimeout ＋ cli stopTimeout 統一 ＋ grace 軸の一本化 ＋ validation co-locate）。**M3 は撤回したため worker resilience PR からは外れる**。C2 は既定値で既発のため優先度高。
4. **PR3 tx retry**: H1（PR0 の `jitter.Full` 利用、Sleeper 結線、`NewTransactionManager` シグネチャ変更＝呼出側 DI 更新、**tx/clock/driver の各 README 契約更新を同一 PR で**）。
5. **PR4 境界**: M1（DB RuntimeParams + REST timeout を **Pre** で）＋ M2（body limit を **Pre** で）。新 httpstack package（`timeout`/`bodylimit`）＋ di extension Pre module（priority 2/3）＋ httpstack README 表追記。
6. **PR5 job timeout**: C1（**hook の OnStop キャンセル化のみ**。boundary/mock/state/cmd 不変）。C2 と同型の OnStop teardown なので PR2 の後だと理解が揃う。波及が hook 1 パッケージに収束するため小さい PR。

各 PR 末尾で `make gen-api`（mock 再生成が要るのは PR1 H2 metric のみ。**rev2 で C1/H1 は mock 再生成不要**＝ boundary 不変）→ `make fix` → `make lint` → `make test`（90% 維持）。コミットは prefix 規約（Fix / Feat / Refactor 等）＋ `Co-Authored-By`、protected branch 直 commit 禁止・push 前確認は CLAUDE.md 準拠。各 README 更新は該当ドキュメントスキル（`sync-readme` / `canonicalize-doc`）経由が望ましい。

---

## 付録: べき論版（rev3 — あるべき構造 / 大改修も厭わない）

rev2 は「blast radius 最小・bug-fix スコープ」で寄せた。本付録は制約を外し「アーキ的に正しい終点」を示す。**前提**: tmp に確定済みの 2 計画が存在し、いずれも `tx.Manager.Do` / inbound idempotency / `httpclient` / `pkg/backoff` の再利用前提で設計済み:
- `tmp/outbox-implementation-plan.ja.md`（Transactional Outbox。D5=ドメイン変更と outbox INSERT を同一 commit / 配信 at-least-once / 冪等 consumer 要求）
- `tmp/idempotency-design.md`（同一 tx claim・Postgres・DTO replay。**実装着手可**）

### べき論の核: 点修正でなく resilience substrate の昇格
known-bugs の真因「**分類は揃うが行動層が無い**」の正しい解は、httpclient を**範＝終点**にせず**再利用可能な共有層へ昇格**し、tx・worker・lifecycle・境界が一様に消費すること。抽出すべき共有抽象は 4 つ:

1. **`pkg/retry`（共有 retry 行動層）**: `classify → bounded attempts → backoff + jitter → budget → deadline-aware` を 1 度だけ実装。`httpclient.doWithRetry` をこの上に載せ替え、H1(tx-retry) も消費。jitter はここに内包し `pkg/backoff` は純粋なまま（Phase 0 の分離方針の上位互換）。
2. **supervised background runner（lifecycle primitive）**: detached goroutine ＋ OnStop キャンセル ＋ grace 内 drain ＋ 起動時 `drain < grace` 検証 を 1 箇所に。job hook / worker hook が両方これを使う。→ **C1 の outlier と C2 の timeout-triad を構造的に解消**（個別配線でなく型で消す）。
3. **end-to-end deadline budget**: request 入口で deadline を 1 点設定し ctx で全層伝播。`statement_timeout` は予算に整合した backstop、`httpclient` は残予算内。独立 magic number（REST_REQUEST_TIMEOUT と DB_STATEMENT_TIMEOUT を別々に置く rev2）を廃し**単一予算から導出**。
4. **port-level redelivery capability**: `Consumer` に遅延再配送を first-class 化（`NackWithBackoff(ctx,m,d)` 等）。engine が policy(exp+jitter from ReceiveCount)を持ち、adapter が native 機構(SQS ChangeMessageVisibility)で honor。

### 項目別: べき論 vs rev2

| 項目 | rev2（最小） | べき論（あるべき） | 反転理由 |
|---|---|---|---|
| **M3** | engine から撤回（adapter/IaC へ） | **port へ昇格**。`Consumer.NackWithBackoff` を追加、engine に policy を戻し、`README.md:31` を改訂 | per-message backoff は circuit breaker と同 class の broker 非依存 policy。IaC 丸投げは「substrate 非依存の resilience」という boilerplate の価値に反する。`Extend` 転用が誤りだっただけで目的は正当 |
| **H1** | `Do` 内 retry ＋ 契約 doc（fn 冪等は呼出側規律） | **outbox と同時に**。tx-retry は `pkg/retry` で実装し、非DB副作用は **outbox row 化して同一 tx に載せる** | doc 依存の規律でなく**構造で**「fn N回再実行 → 外部副作用多重発火」を消す。outbox plan D5 がまさにこの原子点。outbox は確定済 |
| **C1 + C2** | 個別の OnStop 配線（hook ごと） | **supervised runner に統合**し job/worker 共通化。grace は `APP_SHUTDOWN_TIMEOUT` 単一軸へ統一、cli ハードコード 30s と fx default を撤廃 | C1 outlier を型で消す。3つのタイムアウト乱立を 1 軸へ。C1/C2 は同一問題系なので 1 抽象で解くのが正 |
| **M1** | timeout middleware と statement_timeout を独立に | **deadline budget 一本化**（入口 1 点 → 全層伝播、statement_timeout は backstop） | 「境界ごとに期限がある」の終点は独立ノブでなく 1 予算からの導出 |
| **M2** | Pre body limit（priority 2） | **同左**（Pre で正しい）＋ 限界値を profile/route 単位で宣言する余地 | 変更なし。rev2 が既に正しい |
| **L1** | deny-list に CGNAT 追加 | **allowlist へ**。`Profile`/`Registry` が許可ホスト/CIDR を宣言 | deny-list は whack-a-mole。per-downstream profile が既にあるので allowlist が自然 |
| **L2** | `Do` 入口の runtime precondition 検証 | **`httpclient.Request` を型安全化**（`Method` を closed type、`Downstream` 必須をコンストラクタで強制） | runtime チェックを型で消す。報告書の L2 真因「string 別名で型が防げない」を根絶。全 caller 波及 |
| **H2** | Extend 失敗を log+metric 化 | **swallowed-error の一掃**。`_ =` discard を worker/infra 横断で監査し握り潰しを class として除去 | 1 箇所でなく「握り潰し」というカテゴリを潰す |
| **H3** | gracefulStop の error を `errors.Join` | **同左** | 局所 bug。べき論でも同じ |
| **Phase 0** | 新規 `pkg/jitter` | **`pkg/retry` に内包**（backoff は純粋維持） | jitter 単独 pkg より retry 行動層の一部が正 |

### べき論の実装順序（substrate 先・点修正後）
1. **共有抽象**: `pkg/retry`、supervised-runner lifecycle、deadline-budget 規約（decisions.md へ）。
2. **substrate 実装**: outbox（送信側）＋ consumer dedup（inbound idempotency 再利用）。← 既存確定 plan を実行。
3. **これらに載せて点修正**: H1(tx-retry, outbox-safe) / M3(port 昇格) / C1+C2(runner 統合) / M1(budget) / M2 / L1(allowlist) / L2(型安全) / H2(sweep)。H3 は独立に随時。

### 正直なコストと立場
- これは boilerplate の「resilience substrate」章を一段引き上げる**中〜大規模リファクタ**。複数 PR、`README`/`decisions.md`/`architecture.md` 改訂、mock 再生成、全 caller 波及（M3 port / L2 Request 型 / outbox）を伴う。
- 一部は **bug fix を超えた機能追加・設計変更**（M3 port 昇格 / outbox / allowlist / Request 型安全）。known-bugs が「対象外（設計判断）」とした関心を**意図的に取り込む**立場。
- 逆に **H3 / M1(b) ラップ / M2 Pre / Phase0 分離方針**は rev2 と実質同一（べき論でも追加リスクなし）。
- **依存ゲート**: べき論 H1 は outbox 着手が前提。outbox 未着手のまま tx-retry だけ入れると rev2 の「fn 冪等は呼出側規律」へ degrade する（= rev2 が outbox 不在時の正しい妥協解）。

---

## 横断チェックリスト（修正レビューの定型化）

report の指摘どおり、C1/H1/M1/M2 は「**境界に上限・期限・再試行があるか**」の別側面。今後の境界追加時に下記を定型確認にすると同型の穴を最初から塞げる:

- [ ] 再試行: 失敗分類を**消費する**有限リトライ（上限・backoff・jitter）があるか
- [ ] 期限: 実行に deadline が ctx で伝播するか（待ち時間ではなく実行を縛るか）
- [ ] 上限: 入力サイズ・同時実行・応答サイズに上限があるか
- [ ] 失敗の可視化: error を握り潰さず log + metric にしているか
- [ ] 停止順序: teardown が in-flight 完了の内側で完結するか（grace との整合）

参照実装は `internal/infrastructure/httpclient/`（outbound 境界でこの問い全てに回答済み）。
