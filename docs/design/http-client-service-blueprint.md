# Resilient Outbound HTTP Client 作業計画書（boilerplate 整合版）

対象: `tomy-ch/go-boilerplate`
親ドキュメント: [resilient-outbound-http-client.ja.md](./resilient-outbound-http-client.ja.md)（設計の根拠・感度分類はそちら）
本書の役割: 設計ノートを **boilerplate の実装規約に合わせて確定**し、着手順に落とした実行計画。

---

## 0. 確定した設計判断（ロック済み）

| # | 論点 | 決定 |
|---|---|---|
| D-1 | port の置き場所・名前 | `internal/usecase/boundary/httpclient/`、IF 名 **`HTTPClientService`**（boundary の `XxxService` 系統に整合） |
| D-2 | port の型 | **net/http 型を一切露出しない自前型**（`http.Header` / `io.ReadCloser` を使わない）。`Method` / `Header` / `Request` / `Response` / `Downstream` を新規定義 |
| D-3 | エラーモデル | typed struct を捨て、**`apperror` sentinel + `xerrors.Wrap`** に寄せる。新規 sentinel は追加しない（`ErrUnavailable` 等に集約）。**status code → apperror の写像も client の責務**（caller は apperror で分岐, §1.1） |
| D-4 | 時刻/待機抽象 | `clock` パッケージに **`Sleeper` を別 interface で追加**（`Now()` 消費者を巻き込まない＝ISP）。`systemClock` が両方実装 |
| D-5 | downstream キーリング | breaker / metrics / profile / rate-limit のキーを **`Downstream`（論理依存名）に統一** |
| D-6 | body 所有権 | substrate は **`[]byte` で返し `io.ReadCloser` を露出しない**。中間 attempt の body は adapter が drain/close |
| D-7 | retry budget スコープ | **per-downstream**（breaker と同じ `Downstream` キーに相乗り） |
| D-8 | o11y | trace 伝播（B-1）+ RED metrics（B-2）を **adapter に標準搭載**（オプションにしない） |

> D-1 の補足: generic `Do` を port にすると HTTP 概念が usecase に漏れる懸念があったが、D-2（自前型）+ D-3（status→apperror 変換を client 内部に閉じる, §1.1）で漏れを断つ。caller は raw status を見ず apperror で分岐する。per-service の typed boundary を別に作らず、単一の `HTTPClientService` で行く。

---

## 1. boundary 設計（`internal/usecase/boundary/httpclient/`）

```go
//go:generate mockgen -source=$GOFILE -destination=mock/mock_httpclient.gen.go -package=mock_$GOPACKAGE

// Package httpclient は外部 HTTP API への送信側通信を抽象化する boundary。
// usecase はこの interface にのみ依存し、net/http を知らない。
package httpclient

// HTTPClientService は外部 HTTP 通信を担う boundary port。
// resilient（timeout/retry/budget/breaker/o11y）は実装側で完結し、
// usecase は意図（Request）と結果（Response）だけを扱う。
type HTTPClientService interface {
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

**ステータスコードの解釈は HTTPClientService の仕事**。caller は raw status ではなく **`Do` が返すバインド済み apperror で分岐**する。これにより usecase に HTTP status が一切漏れない。

- 2xx → `(resp, nil)`。
- 非 2xx（4xx/5xx）→ `(resp, err)`。`err` は status を写像した **apperror sentinel**（§2 の表）。`resp` も返すので body 等の詳細は参照可能だが、**判断の信号は `err`（バインドされた apperror）**。
- 応答未取得（transport / overall deadline / circuit open / budget 枯渇 / ctx cancel）→ `(nil, err)`。同じく apperror sentinel。

```go
// caller 側の分岐は raw status ではなく apperror で行う。
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
- 実装: `internal/infrastructure/httpclient/errors.go` に「transport 事象 → sentinel」「status code → sentinel」の 2 写像を関数化し、`xerrors.Wrap(apperror.ErrXxx, "...")` で集約。caller には公開しない（client 内部関数）。

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
- DI で `clock.Sleeper` を別 provider として公開。
- `make gen-api` で mock 再生成。
- **影響範囲**: 既存 `clock.Clock` 消費者は無変更（Sleeper は新規 IF のため）。

---

## 4. profile レジストリ（D-5 / M-1）— 仮案

usecase は profile を知らない。infrastructure 内部で `Downstream` をキーに保持する。

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

## 5. o11y 標準搭載（D-8）

adapter constructor が `observability.TracerFactory`（既存 `LayerTracer`）と meter を必須引数で受け、以下を**常時**行う:

- **trace 伝播（B-1）**: `Do` で infra child span を開始し、W3C `traceparent` を outgoing `Header` に inject。span 名は `observability.BuildSpanName(infra, "httpclient", "Do")` 系。
- **metrics（B-2）**: `Downstream` + status class でラベルした RED（request count / latency histogram / error rate）+ retry count + breaker state gauge。
- attempt は親 `Do` span の子 span にして retry が可視化されるようにする。

オプション化しない（引数を nil 許容にしない）。

---

## 6. ファイル構成（新規・変更）

```
internal/usecase/boundary/httpclient/
  httpclient.go          # 新規: HTTPClientService + 自前型（Method/Header/Request/Response/Downstream）
  mock/                  # 新規: 生成 mock
internal/usecase/boundary/clock/
  clock.go               # 変更: Sleeper 追加
  mock/                  # 変更: 再生成
internal/infrastructure/system/
  clock.go               # 変更: Sleep 実装
internal/infrastructure/httpclient/
  client.go              # 新規: HTTPClientService 実装（transport + Do + retry ループ）
  retry.go               # 新規: 分類(A-3) / backoff+jitter(A-4) / deadline 規律(A-1)
  budget.go              # 新規: retry budget per-downstream(A-5)
  breaker.go             # 新規: circuit breaker per-downstream(A-6)
  profile.go             # 新規: Profile / BreakerConfig / Registry(M-1)
  errors.go              # 新規: transport 事象 + status code → apperror sentinel(D-3)
  observability.go       # 新規: span + metrics(D-8)
internal/di/module/
  infrastructure.go      # 変更: httpclient.New / clock.Sleeper を fx.Provide
```

> apperror: 変更なし（既存 sentinel に集約）。将来 circuit-open を区別する必要が出たら `ErrCircuitOpen` 追加を別途検討。

---

## 7. 実装フェーズ（各フェーズ末で compile + test 通過）

| Phase | 内容 | 完了条件 |
|---|---|---|
| **P0 境界拡張** | clock `Sleeper` 追加 + `systemClock.Sleep` 実装 + `make gen-api` で mock 再生成 | 既存 test green、Sleeper mock 生成 |
| **P1 boundary** | `httpclient` パッケージ（IF + 自前型）+ mock。usecase 未接続 | パッケージ単体で compile、型定義の sanity test |
| **P2 adapter 骨格** | transport 構築 + 単発 `Do`（retry 無し）+ body `[]byte` 読み切り（MaxBytesReader/drain・close）+ errors.go（transport + status→apperror 写像）+ o11y span/metrics（D-8） | httptest で 2xx→`(resp,nil)` / 4xx・5xx→対応 apperror / transport 失敗→`ErrUnavailable`（table-driven で全 status 写像を検証） |
| **P3 retry コア** | 分類(A-3) + backoff/jitter(A-4, `Sleeper` 注入) + deadline 規律(A-1) + 冪等性ガード(A-2) | §8 の retry/冪等/deadline/backoff テスト green |
| **P4 budget + breaker** | retry budget(A-5/D-7) + circuit breaker(A-6) per-downstream | budget 枯渇・breaker open→fail-fast→half-open のテスト green |
| **P5 profile + DI** | Registry(M-1) + `fx.Provide` 配線 + デフォルト profile 供給（仮案） | DI 起動 test、未登録キー fallback |
| **P6 推奨群** | Retry-After(B-7) + redaction(B-3) + metrics 拡充(B-2) | 各単体テスト |

> 各フェーズは「コンパイル可能・テスト green」を維持して進める（big-bang にしない）。

---

## 8. テスト戦略（設計ノート §7 を boilerplate 規約で）

- 全 `t.Parallel()`、`t.Run` 分割、ケース名は日本語、`正常系`/`異常系` を最上位グループに。
- error 系は `require`、終端値は `assert`（testifylint require-error）。
- 生成 mock（`clock.Sleeper` mock 等）を使い、手書き mock を作らない。
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

1. breaker / backoff を自前実装か外部ライブラリ（`sony/gobreaker` / `cenkalti/backoff`）か。**adapter 内部に閉じ port には漏らさない**前提なら依存追加可。P4 着手時に判断。
2. profile 供給を typed config に移す閾値（downstream 数）。M-1 のキー固定により後追い変更可。
3. C-1 auth 自動更新 / C-2 client-side rate limit / C-3 mTLS / C-4 SSRF 防御 は条件付き。`Profile` に差込口だけ用意し、要件発生時に実装。
4. Idempotency-Key の生成・再利用規約（同一論理操作の retry 間で同じ key）をどの層が保証するか。inbound idempotency（`idempotency_keys` テーブル）と対称。

---

## 10. 着手前チェック（CLAUDE.md 準拠）

- [ ] 本計画を user 承認
- [ ] feature ブランチ作成（`release/*` から）
- [ ] P0→P6 を順に、各フェーズで `make fix` → `make test`
- [ ] 生成物（mock）は `make gen-api`、手編集しない
- [ ] 可視出力・コメント・テスト名は日本語
