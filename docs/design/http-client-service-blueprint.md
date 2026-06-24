# Resilient Outbound HTTP Client 作業計画書（boilerplate 整合版）

対象: `tomy-ch/go-boilerplate`
親ドキュメント: [resilient-outbound-http-client.ja.md](./resilient-outbound-http-client.ja.md)（設計の根拠・感度分類はそちら）
本書の役割: 設計ノートを **boilerplate の実装規約に合わせて確定**し、着手順に落とした実行計画。

> 改訂（ローカルレビュー反映）: ① **配置を db.driver 流（A案）に変更** — 汎用 resilient client は **infra 内部 substrate**（`internal/infrastructure/httpclient/`、`rdb/driver` 相当）とし、usecase は **意味的 gateway IF** に依存（boundary に汎用 HTTP port は置かない）。② trace は **otelhttp 自動計装に委譲**（`otelpgx` / `otelecho` と対称、手動 inject 廃止）。③ RED metrics は **`internal/observability` 側 struct**（`WorkerMetrics` 対称）。④ 意味的 IF は戻り値で使い分け（§6.2、状況次第）— **Domain は既存 `domain.Repository` を HTTP 実装**（新概念なし）、**DTO は boundary gateway**（`auth` 流）。"gateway" は DTO モード専用語。⑤ 層境界は既存 depguard で担保（usecase/domain/controller は infra import 不可）。infra の `net/http` deny は「infra でハンドラを作るな」の巻き添えなので、outbound substrate のみ `.golangci-full.yaml` で除外する（§6.1）。

---

## 0. 確定した設計判断（ロック済み）

| # | 論点 | 決定 |
|---|---|---|
| D-1 | port の置き場所・名前（**db.driver 流 / A案**） | 汎用 resilient client は **infra 内部 substrate**（`internal/infrastructure/httpclient/`、`rdb/driver.DatabaseDriver` 相当）。IF 名は `httpclient.Client`（infra-internal）。**boundary には汎用 HTTP port を置かない**。usecase 直依存は**外部サービスごとの意味的 gateway IF**（DB の `domain.Repository` 相当、`auth.Authenticator` と同列）。判断基準は「usecase が直接 import するか」＝ driver が infra にあるのと同じ理屈。gateway IF の戻り値 / 配置（DTO / Domain）は状況次第＝§6.2 |
| D-2 | substrate の型 | substrate API は **net/http 型を露出しない自前型**（`Method` / `Header` / `Request` / `Response` / `Downstream`）。`Body []byte`（retry replay 可）/ `io.ReadCloser` 非露出。これらは **infra-internal**（gateway が意味的型へ変換し usecase に渡す。usecase は substrate 型を見ない） |
| D-3 | エラーモデル | typed struct を捨て、**`apperror` sentinel + `xerrors.Wrap`** に寄せる。新規 sentinel は追加しない（`ErrUnavailable` 等に集約）。**status code → apperror の写像も client の責務**（caller は apperror で分岐, §1.1） |
| D-4 | 時刻/待機抽象 | `clock` パッケージに **`Sleeper` を別 interface で追加**（`Now()` 消費者を巻き込まない＝ISP）。`systemClock` が両方実装 |
| D-5 | downstream キーリング | breaker / metrics / profile / rate-limit のキーを **`Downstream`（論理依存名）に統一** |
| D-6 | body 所有権 | substrate は **`[]byte` で返し `io.ReadCloser` を露出しない**。中間 attempt の body は adapter が drain/close |
| D-7 | retry budget スコープ | **per-downstream**（breaker と同じ `Downstream` キーに相乗り） |
| D-8 | o11y | **trace は otelhttp 自動計装に委譲**（`otelpgx` / `otelecho` と対称＝transport 自動計装 + 各層 LayerTracer span）。手動 traceparent inject はしない。**RED metrics（B-2）は `internal/observability` 側に struct 配置**（`WorkerMetrics` 対称）。両者ともオプションにしない（§5） |

> D-1 の補足（A案の根拠）: boundary の membership 基準は「技術 vs 外部」ではなく **「usecase が直接 import するか」**。`clock.Clock` が boundary なのは usecase が `Now()` を直に呼ぶから、`driver.DatabaseDriver` が infra なのは usecase が呼ばない（`domain.Repository` の2ホップ奥）から。汎用 HTTP client も usecase は直接叩かず **意味的 gateway 経由**で使う。よって client = infra substrate（driver 相当）、gateway IF = usecase 直依存ポート（repository が実装する `domain.Repository` 相当）。これで DB と HTTP が対称になり、boundary を「usecase 直依存ポートの集合」に保てる。status→apperror 写像（D-3）は substrate 内部に閉じ、gateway 経由で apperror が usecase へ伝播する。

---

## 1. substrate 設計（`internal/infrastructure/httpclient/` — driver 相当・infra 内部）

```go
//go:generate mockgen -source=$GOFILE -destination=mock/mock_httpclient.gen.go -package=mock_$GOPACKAGE

// Package httpclient は外部 HTTP 通信の resilient な substrate（driver 相当）を提供する。
// gateway 実装がこれに依存し、usecase はこの package を知らない（rdb/driver と同じ立ち位置）。
package httpclient

// Client は resilient な外部 HTTP 通信の substrate port（infra 内部・driver.DatabaseDriver 相当）。
// resilient（timeout/retry/budget/breaker/o11y）は実装側で完結し、
// gateway は意図（Request）と結果（Response）だけを扱う。
type Client interface {
	Do(ctx context.Context, req *Request) (*Response, error)
}

// Downstream は breaker / metrics / profile / budget の共通キー（論理依存名）。D-5
type Downstream string

// Method は HTTP メソッド（net/http に依存しない自前型）。D-2
type Method string

const (
	MethodGet    Method = "GET"
	MethodPost   Method = "POST"
	MethodPut    Method = "PUT"
	MethodPatch  Method = "PATCH"
	MethodDelete Method = "DELETE"
)

// Header は net/http.Header を露出しないための自前型。D-2
type Header map[string][]string

// Request は 1 呼び出しの意図を明示する。
type Request struct {
	Downstream     Downstream // 必須。キーが自然に揃う
	Method         Method
	URL            string
	Header         Header
	Body           []byte // バッファ済み = retry で replay 可能（stream は非対象）
	IdempotencyKey string // 非冪等メソッドを retry 安全にする（任意）
	AllowRetry     bool   // POST 等を明示的に retry 許可。後述の不変条件あり
}

// Response は io.ReadCloser を露出しない。D-6
type Response struct {
	StatusCode int
	Header     Header
	Body       []byte // adapter が MaxBytesReader で読み切り済み
}
```

### 1.1 `Do` の戻り値方針 — status 処理は client の責務（確定）

**ステータスコードの解釈は substrate（`httpclient.Client`）の仕事**。直接の caller は gateway 実装で、raw status ではなく **`Do` が返すバインド済み apperror** を受け取り、必要なら意味的エラーへ翻訳して usecase へ返す。これにより usecase に HTTP status が一切漏れない。

- 2xx → `(resp, nil)`。
- 非 2xx（4xx/5xx）→ `(resp, err)`。`err` は status を写像した **apperror sentinel**（§2 の表）。`resp` も返すので body 等の詳細は参照可能だが、**判断の信号は `err`（バインドされた apperror）**。
- 応答未取得（transport / overall deadline / circuit open / budget 枯渇 / ctx cancel）→ `(nil, err)`。同じく apperror sentinel。

```go
// 実装側（Repository / gateway）の分岐は raw status ではなく apperror で行う。
resp, err := client.Do(ctx, req)
switch {
case xerrors.Is(err, apperror.ErrNotFound):       // downstream 404
case xerrors.Is(err, apperror.ErrTooManyRequests): // downstream 429
case xerrors.Is(err, apperror.ErrUnavailable):     // 5xx / transport / circuit / budget
case err != nil:                                   // その他
default:                                           // 2xx
}
```

> status→apperror の写像は **client 内部（`errors.go`）に閉じる**。caller 向けの `StatusError` ヘルパは作らない（caller に status 解釈を持たせない）。
> 「外部 503 を自 API でどう表現するか（500 として返す / 503 透過 等）」の最終決定は caller の責務だが、その判断は **`Do` が返した apperror（例: `ErrUnavailable`）を起点**に行う。`resp.StatusCode` は診断・ログ用に保持するだけで、分岐の一次信号にはしない。

---

## 2. エラーマッピング（D-3）— client が status まで写像 / 新規 sentinel 追加なし

`Do` が返す `err` は、transport 事象と HTTP status の両方を **client 内部で apperror sentinel に統一**したもの。

| 事象 | 返す sentinel | 備考 |
|---|---|---|
| network / DNS / TLS 失敗 | `ErrUnavailable` | wrap msg に原因 |
| per-attempt / overall deadline | `ErrUnavailable` | 「retry 尽きた」も含む |
| circuit open | `ErrUnavailable` | msg `"circuit open: <downstream>"`。区別が要れば将来 `ErrCircuitOpen` 追加 |
| retry budget 枯渇 | `ErrUnavailable` | msg `"retry budget exhausted"` |
| ctx cancel | `ErrCanceled` | |
| HTTP 400 | `ErrInvalidArgument` | |
| HTTP 401 | `ErrUnauthenticated` | |
| HTTP 403 | `ErrPermissionDenied` | |
| HTTP 404 | `ErrNotFound` | |
| HTTP 409 | `ErrConflict` | |
| HTTP 422 | `ErrValidation` | |
| HTTP 429 | `ErrTooManyRequests` | |
| HTTP 5xx | `ErrUnavailable` | retry 尽きた最終結果 |
| その他 4xx | `ErrInvalidArgument` | default |

- 非 2xx でも `resp`（status / header / body）は返す（診断・ログ用）。**caller の分岐は `err` の sentinel で行う**（§1.1）。
- 実装: `internal/infrastructure/httpclient/errors.go` に「transport 事象 → sentinel」「status code → sentinel」の 2 写像を関数化し、`xerrors.Wrap(apperror.ErrXxx, "...")` で集約。caller には公開しない（client 内部関数）。位置づけは RDB の `pgerror.NormalizeError`（DB エラー → apperror）の HTTP 版。

---

## 3. clock 境界拡張（D-4）

```go
// internal/usecase/boundary/clock/clock.go に追加（Now() は不変）
//go:generate mockgen -source=$GOFILE -destination=mock/mock_clock.gen.go -package=mock_$GOPACKAGE

// Sleeper は決定的テスト可能な待機を提供する。backoff / breaker timer が依存。
type Sleeper interface {
	// Sleep は d 経過まで待機する。ctx が先に done になれば即座に ctx.Err() を返す。
	Sleep(ctx context.Context, d time.Duration) error
}
```

- `internal/infrastructure/system/clock.go` の `systemClock` に `Sleep` を実装（`time.NewTimer` + `select { ctx.Done() / timer.C }`）。同一 struct が `Clock` と `Sleeper` の両方を満たす。
- DI で `clock.Sleeper` を別 provider として公開する。既存 `NewClock` は IF 型（`clock.Clock`）を返すため fx から `Sleeper` を導出できない。**`system.NewSleeper() clock.Sleeper` を新設**し、clock module に `fx.Provide(system.NewSleeper)` を追加する。
- `make gen-api` で mock 再生成（同一 `clock.go` から `Clock` / `Sleeper` 両 mock が生成される）。
- **影響範囲**: 既存 `clock.Clock` 消費者は無変更（Sleeper は新規 IF のため）。

---

## 4. profile レジストリ（D-5 / M-1）— 仮案

usecase も gateway も profile を意識しない。substrate（infra）内部で `Downstream`（gateway が自分の論理依存名を渡す）をキーに保持する。

```go
// internal/infrastructure/httpclient/profile.go（infra 内部・port に漏らさない）
type Profile struct {
	PerAttemptTimeout time.Duration // A-1
	OverallTimeout    time.Duration // A-1
	MaxAttempts       int           // A-4
	BaseBackoff       time.Duration // A-4
	MaxBackoff        time.Duration // A-4
	RetryBudgetRatio  float64       // A-5 / D-7
	MaxResponseBytes  int64         // B-6
	Breaker           BreakerConfig // A-6
	// RateLimit *RateConfig  // C-2（任意・後続）
	// Auth      AuthProvider // C-1（任意・後続）
}

type BreakerConfig struct {
	FailureThreshold float64       // open に倒す失敗率
	MinRequests      int           // 評価に必要な最小サンプル数
	OpenDuration     time.Duration // open→half-open までの待機
	HalfOpenProbes   int           // half-open で通すプローブ数
}

// Registry は Downstream → Profile。未登録キーは安全側デフォルトに fallback。
type Registry interface {
	Profile(d httpclient.Downstream) Profile
}
```

供給方法（**仮案・後で変更可**）: まずはコード定義のデフォルト + 1〜2 downstream を登録。downstream 数が増えたら typed config 構造体へ移す。キー（`Downstream`）だけ先に固定しておけば供給方法の変更は作り直しにならない。

---

## 5. o11y 標準搭載（D-8）— trace は otelhttp、metrics は observability struct

このリポジトリの通信境界は **「transport/driver レベルの自動計装（otel-contrib）＋各層 LayerTracer span」** で統一されている（inbound=`otelecho`、DB=`otelpgx`）。outbound HTTP もこれに揃える。

trace span は **2 層**（DB と対称）: 意味的 IF 実装（Domain なら Repository / DTO なら gateway）が層 span を張り、substrate（driver 相当）の otelhttp が HTTP span を張る。

### 5.1 trace（otelhttp 自動計装に委譲）

- substrate の `http.Client` の `Transport` を **`otelhttp.NewTransport(base)` でラップ**する（DB の `pgxpool` に `otelpgx` を結線するのと対称）。これにより **HTTP span 生成と W3C `traceparent` の outgoing inject が自動**で行われる（global propagator は `provider.go` で W3C TraceContext + Baggage 登録済み）。
- 手動の traceparent inject は**行わない**。`internal/observability/propagation.go` への inject ヘルパ追加は**不要**。
- **substrate 自身は LayerTracer span を張らない**（`driver` が otelpgx に任せ自前 span を持たないのと対称）。retry の各 attempt は otelhttp が RoundTrip ごとに子 span 化する。
- **層 span は意味的 IF 実装が張る**: Repository / gateway 実装が `tf.Infra()` の `LayerTracer` で意味的操作に層 span を張り、その配下に otelhttp の attempt span がぶら下がる（trace 木: usecase span → 層 span → otelhttp attempt span ×N）。`TracerFactory` は **その実装（Repository / gateway）の constructor** が受ける（substrate は受けない）。

### 5.2 metrics（observability 側 struct）

- RED metrics（request count / latency histogram / error rate）+ retry count + breaker state gauge を、`Downstream` + status class でラベルし計上する。担当は **substrate**（retry / breaker / status を知るのは substrate のため）。
- 配置は **`internal/observability/` の `HTTPClientMetrics`**（`WorkerMetrics` と対称。`metric.MeterProvider` 注入、`NewNoopHTTPClientMetrics` テスト支援も用意）。substrate constructor がこの struct を必須引数で受ける。
- **otelhttp 自動 metrics（`http.client.*`）は無効化**し、二重計上を避ける（otelhttp は span のみ利用、`Downstream` 単位の RED は `HTTPClientMetrics` が持つ）。

trace・metrics いずれもオプション化しない（引数を nil 許容にしない）。

> 構成まとめ: **substrate constructor** = `clock.Sleeper` + `Registry` + `HTTPClientMetrics` + otelhttp ラップ transport（`TracerFactory` 不要）。**意味的 IF 実装（Repository / gateway）の constructor** = `httpclient.Client` + `observability.TracerFactory`（層 span 用）。
> 依存追加: `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp`（contrib 系は `otelecho` / `otelpgx` / `runtime` 等で既に複数導入済み）。

---

## 6. ファイル構成（新規・変更）

```
# ── substrate（driver 相当・infra 内部）──
internal/infrastructure/httpclient/
  httpclient.go          # 新規: Client IF + 自前型（Method/Header/Request/Response/Downstream）
  mock/                  # 新規: 生成 mock（gateway テストが substrate を差し替える用）
  client.go              # 新規: Client 実装（transport=otelhttp ラップ + Do + retry ループ）
  retry.go               # 新規: 分類(A-3) / backoff+jitter(A-4) / deadline 規律(A-1)
  budget.go              # 新規: retry budget per-downstream(A-5)
  breaker.go             # 新規: circuit breaker per-downstream(A-6)
  profile.go             # 新規: Profile / BreakerConfig / Registry(M-1)
  errors.go              # 新規: transport 事象 + status code → apperror sentinel(D-3)
# ── 意味的 IF（サンプルは DTO モード=gateway を1つ。Domain モードは domain.Repository を使う・§6.2）──
internal/usecase/boundary/<service>/       # 意味的 gateway IF（DTO モード。usecase テスト用 mock）
  <service>.go           # 新規: Gateway IF（意味的メソッド）
  mock/                  # 新規: 生成 mock（usecase テスト用）
internal/infrastructure/external/<service>/  # 意味的 IF 実装（httpclient.Client を使用・層 span を張る）
  <service>_gateway.go   # 新規: 実装（Request 組立 → client.Do → 意味的型/エラーへ変換。Domain モードなら <agg>_repository.go）
# ── 境界拡張・o11y・DI ──
internal/usecase/boundary/clock/
  clock.go               # 変更: Sleeper 追加
  mock/                  # 変更: 再生成（Clock / Sleeper 両 mock）
internal/infrastructure/system/
  clock.go               # 変更: Sleep 実装 + NewSleeper() 追加
internal/observability/
  http_client_metrics.go # 新規: HTTPClientMetrics（RED + retry count + breaker gauge。WorkerMetrics 対称）
internal/di/module/
  infrastructure.go      # 変更: httpclient.New（substrate）/ <service>.New（gateway）/ system.NewSleeper を fx.Provide
  observability.go       # 変更: observability.NewHTTPClientMetrics を fx.Provide
```

> apperror: 変更なし（既存 sentinel に集約）。将来 circuit-open を区別する必要が出たら `ErrCircuitOpen` 追加を別途検討。

### 6.1 層境界 enforcement（lint）

substrate を infra に置くことで、層境界は **既存 depguard でそのまま担保される**（追加機構は最小）。

- **「infra 以外から呼べない」保証（追加機構不要）**: `.golangci-full.yaml` の `maintain_a_sound_domain` / `maintain_a_sound_usecase` / `maintain_a_sound_controller` が `go-boilerplate/internal/infrastructure/` を deny 済み。よって substrate（`internal/infrastructure/httpclient/`）は **usecase / domain / controller から import 不可**。gateway 実装は同じ infra 層なので import 可。db.driver と同じく「呼べるのは infra（と composition root の di / cli / cmd）だけ」。
- **net/http の許可（outbound 例外）**: `maintain_a_sound_infrastructure` の `net/http` deny の本来の趣旨は **「infra で HTTP ハンドラを勝手に定義させない」**（server 側＝ハンドラは controller の責務）。net/http の全面 deny はその**おまけ（巻き添え）**で、ハンドラを作らない outbound client の用途まで巻き込んでいるだけ。**outbound substrate はハンドラを定義しない**ので、deny から除外して問題ない。
  - 方法: `maintain_a_sound_infrastructure` の `files` に `!**/internal/infrastructure/httpclient/**.go` を追加して全体ルールから外し、substrate 専用ルール（`files: **/internal/infrastructure/httpclient/**.go`）で echo / controller の deny のみ再掲（net/http は許可＝多層防御を維持）。書式は既存の uuid / otelecho 例外（`!**/...` 行）に倣う。コメントで inbound（禁止）/ outbound（許可）を明記。
  - 変更は **`.golangci-full.yaml` のみ**（軽量 `.golangci.yaml` は infra の net/http を禁止していないため不要）。
  - `otelhttp`（`go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp`）は `net/http` と別 import パスのため deny に当たらない。

### 6.2 意味的 IF の選択 — Domain は Repository / DTO は gateway（状況次第）

substrate を使う「意味的 IF」は、戻り値が Domain か DTO かで **既存の抽象をそのまま使い分ける（新概念を増やさない）**。

| モード | 戻り値型 | 意味的 IF | 配置（IF / 実装） | 例 |
|---|---|---|---|---|
| **Domain モード** | domain entity / VO | **`domain.Repository`**（DB 版と同一抽象。永続先が HTTP なだけ） | `domain/<agg>/<agg>_repository.go` / `infra/external/<agg>/<agg>_repository.go` | 外部が `Customer` の正本／外部 `CreditScore` を与信ロジックが使う |
| **DTO モード** | 自前 DTO / VO | **boundary gateway IF**（`auth.Authenticator` と同列） | `usecase/boundary/<service>/` / `infra/external/<service>/<service>_gateway.go` | 通知送信・トークン検証・為替取得で素通し |

- **Domain モードは新概念を作らない**: 「外部データを domain として持つ」なら **既存の `domain.Repository` を定義して HTTP 実装するだけ**。usecase から見れば DB-backed か HTTP-backed かは実装詳細で、`domain.<Agg>Repository` に依存する体験は完全に同じ（storage-agnostic Repository）。**"gateway" という語は DTO モード専用**とする。
- **判別軸**: データの出所（外部であること）ではなく **「業務ルールがそれを不変条件付きの中核概念として扱うか」**。扱う→Repository（Domain）、素通し→gateway（DTO）。
- **状況次第＝同一外部サービスでもオペレーション / ユースケース単位で変わりうる**。一方に固定せず、IF 単位で判別軸に基づき選ぶ。
- **ACL（Domain モード時の規律）**: 外部 JSON を直接 domain にデシリアライズしない。実装が外部 payload → `domain.New...()` で不変条件を通して構築する（repository の row→entity と同じ）。外部 API の都合を domain に漏らさない防壁。
- **過剰 domain 化の禁止**: 「持てるから持つ」をやらない。中核業務概念でないものを Repository 化すると外部モデル / ライフサイクルに domain が結合する（DDD の典型失敗）。
- 両モードとも **実装は `infra/external/<name>/`**（substrate 利用・`tf.Infra()` 層 span を張る）。**モードで変わるのは IF の種別（Repository / gateway）と戻り値型だけ**。

---

## 7. 実装フェーズ（各フェーズ末で compile + test 通過）

| Phase | 内容 | 完了条件 |
|---|---|---|
| **P0 境界拡張** | clock `Sleeper` 追加 + `systemClock.Sleep` 実装 + `NewSleeper` provider + `make gen-api` で mock 再生成 | 既存 test green、Sleeper mock 生成 |
| **P1 substrate 型/IF** | `internal/infrastructure/httpclient` パッケージ（`Client` IF + 自前型）+ mock。gateway 未接続 | パッケージ単体で compile、型定義の sanity test |
| **P2 substrate 骨格** | transport 構築（otelhttp ラップ）+ 単発 `Do`（retry 無し）+ body `[]byte` 読み切り（MaxBytesReader/drain・close）+ errors.go（transport + status→apperror 写像）+ metrics（D-8 5.2）。**net/http 初出のため `.golangci-full.yaml` に outbound 例外を追加（§6.1）** | httptest で 2xx→`(resp,nil)` / 4xx・5xx→対応 apperror / transport 失敗→`ErrUnavailable`（table-driven で全 status 写像を検証）+ `make lint`（full）green |
| **P3 retry コア** | 分類(A-3) + backoff/jitter(A-4, `Sleeper` 注入) + deadline 規律(A-1) + 冪等性ガード(A-2) | §8 の retry/冪等/deadline/backoff テスト green |
| **P4 budget + breaker** | retry budget(A-5/D-7) + circuit breaker(A-6) per-downstream | budget 枯渇・breaker open→fail-fast→half-open のテスト green |
| **P5 profile + substrate DI** | Registry(M-1) + `fx.Provide(httpclient.New)` 配線 + デフォルト profile 供給（仮案） | DI 起動 test、未登録キー fallback |
| **P6 意味的 IF サンプル** | サンプルは DTO モード=gateway IF（boundary）+ 実装（`infra/external`、`tf.Infra()` 層 span + `client.Do` + 意味的型/エラー変換）+ mock 生成 + DI 配線 + usecase 利用サンプル（Domain モードは既存 `domain.Repository` を HTTP 実装するだけ・§6.2） | 実装は httptest、usecase は意味的 IF の mock でテスト green |
| **P7 推奨群** | Retry-After(B-7) + redaction(B-3) + metrics 拡充(B-2) | 各単体テスト |

> 各フェーズは「コンパイル可能・テスト green」を維持して進める（big-bang にしない）。

---

## 8. テスト戦略（設計ノート §7 を boilerplate 規約で）

- 全 `t.Parallel()`、`t.Run` 分割、ケース名は日本語、`正常系`/`異常系` を最上位グループに。
- error 系は `require`、終端値は `assert`（testifylint require-error）。
- 生成 mock（`clock.Sleeper` mock 等）を使い、手書き mock を作らない。
- **テスト階層（DB と対称）**: substrate（`httpclient`）は httptest で transport を差し替え検証。意味的 IF 実装（Repository / gateway）も httptest（substrate を実物で通す＝repository が実 DB を通すのと同じ）。usecase は **意味的 IF の mock を注入**（Domain モードは `domain.Repository` mock、DTO モードは gateway mock）。usecase は substrate を一切知らない。
- **retry 分類**: 4xx は retry されない / 5xx・429・network は retry される（httptest で transport 差し替え）。
- **冪等性ガード**: `AllowRetry=false` の POST が 1 回しか飛ばない。
- **AllowRetry 不変条件**: `AllowRetry=true` かつ `IdempotencyKey==""` を**コンストラクタ/`Do` 入口で拒否**（二重課金防止。設計ノート A-2 の格上げ）。
- **deadline 規律**: caller ctx を短くすると retry が deadline を超えない。
- **budget / breaker**: 連続失敗で budget 枯渇・breaker open、open 中は即 fail-fast（`ErrUnavailable`）。
- **backoff**: `Sleeper` mock を注入し待機列が expo + full jitter 範囲（決定的）。
- **body**: 上限超過が `MaxBytesReader` で打ち切られ、中間 attempt の close 漏れが無い（D-6）。
- 新規/変更パッケージは **90% 超**（CLAUDE.md）。

---

## 9. 未決事項（着手をブロックしない・実装中に確定）

0. **意味的 IF の戻り値 / 配置（Domain か DTO か）**: **§6.2 で確定** — 状況次第の2モード（Domain→既存 `domain.Repository` を HTTP 実装、DTO→`usecase/boundary/<service>/` gateway）。判別軸＝業務ルールが不変条件付きの中核概念として扱うか。substrate 実装（P1〜P5）はこの論点に非依存で先行可能。なお `auth` 等を「外部呼び出し系」として名前空間グルーピングし直すかは別途（必須でない・現状フラット維持で可）。
1. breaker / backoff を自前実装か外部ライブラリ（`sony/gobreaker` / `cenkalti/backoff`）か。**adapter 内部に閉じ port には漏らさない**前提なら依存追加可。P4 着手時に判断。
2. profile 供給を typed config に移す閾値（downstream 数）。M-1 のキー固定により後追い変更可。
3. C-1 auth 自動更新 / C-2 client-side rate limit / C-3 mTLS / C-4 SSRF 防御 は条件付き。`Profile` に差込口だけ用意し、要件発生時に実装。
4. Idempotency-Key の生成・再利用規約（同一論理操作の retry 間で同じ key）をどの層が保証するか。inbound idempotency（`idempotency_keys` テーブル）と対称。

---

## 10. 着手前チェック（CLAUDE.md 準拠）

- [x] 本計画を user 承認（ローカルレビュー反映済み: **配置=db.driver 流 A案**（汎用 client は infra substrate / usecase は gateway IF）/ gateway 戻り値は DTO・Domain を状況次第（§6.2）/ trace=otelhttp / metrics=observability / 層境界は depguard 担保（§6.1））
- [x] feature ブランチ作成（`release/*` から）— 実装用 `feature/http-client-service-impl`（`release/v1.5.0` 起点）
- [ ] P0→P6 を順に、各フェーズで `make fix` → `make test`
- [ ] 生成物（mock）は `make gen-api`、手編集しない
- [ ] 可視出力・コメント・テスト名は日本語
