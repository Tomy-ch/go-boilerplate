//go:generate mockgen -source=$GOFILE -destination=mock/mock_httpclient.gen.go -package=mock_$GOPACKAGE

// Package httpclient は、外部 HTTP 通信の resilient な substrate（driver 相当）を提供します。
//
// gateway / Repository などの意味的 IF 実装がこの substrate に依存し、usecase はこの
// パッケージを直接知りません（rdb/driver と同じ立ち位置）。timeout / retry / budget /
// breaker / o11y といった resilient な振る舞いは実装側で完結し、呼び出し側は意図
// （Request）と結果（Response）だけを扱います。
//
// net/http に依存しない自前型（Method / Header / Request / Response / Downstream）を
// 公開し、ステータスコードの解釈や apperror への写像は substrate 内部で完結させます。
package httpclient

import "context"

// HTTP メソッドの定数群です。Method は閉じた型なので、これら以外の不正な
// メソッド文字列は型レベルで構築できません（L2: string 別名による型の穴を根絶）。
var (
	// MethodGet は、HTTP GET メソッドです。
	MethodGet = Method{"GET"}
	// MethodPost は、HTTP POST メソッドです。
	MethodPost = Method{"POST"}
	// MethodPut は、HTTP PUT メソッドです。
	MethodPut = Method{"PUT"}
	// MethodPatch は、HTTP PATCH メソッドです。
	MethodPatch = Method{"PATCH"}
	// MethodDelete は、HTTP DELETE メソッドです。
	MethodDelete = Method{"DELETE"}
)

// Client は、resilient な外部 HTTP 通信の substrate port です（infra 内部・driver.DatabaseDriver 相当）。
type Client interface {
	// Do は、req を送信し Response を返します。
	//
	// 2xx は (resp, nil)、非 2xx は (resp, apperror) を返します（resp は診断用に保持）。
	// 応答を取得できなかった場合（transport 失敗 / deadline 超過 / circuit open /
	// budget 枯渇 / ctx cancel）は (nil, apperror) を返します。
	// 呼び出し側の分岐は raw status ではなく返り値の apperror sentinel で行います。
	Do(ctx context.Context, req *Request) (*Response, error)
}

// Downstream は、breaker / metrics / profile / budget の共通キー（論理依存名）です。
type Downstream string

// Method は、HTTP メソッドを表す閉じた自前型です（net/http に依存しません）。
//
// 内部フィールドが非公開なため、パッケージ外からは MethodGet 等の定義済み定数しか
// 構築できません。`Method("garbage")` のような不正な文字列キャストは型レベルで弾かれます
// （L2: 真因「string 別名で型が防げない」を根絶）。
type Method struct{ s string }

// Header は、HTTP ヘッダを表す自前型です（net/http.Header を露出しません）。
type Header map[string][]string

// Request は、1 回の外部 HTTP 呼び出しの意図を表します。
type Request struct {
	// Downstream は、論理依存名です（必須）。breaker / metrics / profile / budget のキーになります。
	Downstream Downstream
	// Method は、HTTP メソッドです。
	Method Method
	// URL は、リクエスト先 URL です。
	// 認証情報などの機密はクエリではなく Header に載せてください（URL は o11y span へ記録されます）。
	URL string
	// Header は、リクエストヘッダです。
	Header Header
	// Body は、バッファ済みのリクエストボディです。retry で replay 可能です（stream は非対象）。
	Body []byte
	// IdempotencyKey は、非冪等メソッドを retry 安全にするためのキーです（任意）。
	IdempotencyKey string
	// AllowRetry は、POST 等の非冪等メソッドを明示的に retry 許可するフラグです。
	// true の場合は IdempotencyKey が必須です（二重実行防止）。
	AllowRetry bool
}

// RequestOption は、NewRequest の任意項目を設定する関数オプションです。
type RequestOption func(*Request)

// Response は、外部 HTTP 呼び出しの結果を表します（io.ReadCloser を露出しません）。
type Response struct {
	// StatusCode は、HTTP ステータスコードです。
	StatusCode int
	// Header は、レスポンスヘッダです。
	Header Header
	// Body は、読み切り済みのレスポンスボディです（上限まで読み込み済み）。
	Body []byte
}

// String は、HTTP メソッド文字列を返します（ゼロ値は ""）。
func (m Method) String() string { return m.s }

// NewRequest は、必須項目（method / downstream / url）を強制してリクエストを生成します（L2）。
// method は閉じた Method 型、downstream は引数で必須化されるため、不正なメソッドや
// downstream 欠落を構築時点で型・シグネチャにより排除します。
func NewRequest(method Method, downstream Downstream, url string, opts ...RequestOption) *Request {
	r := &Request{
		Downstream: downstream,
		Method:     method,
		URL:        url,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// WithHeader は、リクエストヘッダを設定します。
func WithHeader(h Header) RequestOption { return func(r *Request) { r.Header = h } }

// WithBody は、リクエストボディを設定します。
func WithBody(b []byte) RequestOption { return func(r *Request) { r.Body = b } }

// WithIdempotencyKey は、冪等性キーを設定します（retry は許可しません）。
func WithIdempotencyKey(key string) RequestOption {
	return func(r *Request) { r.IdempotencyKey = key }
}

// WithRetry は、非冪等メソッドの retry を許可します。IdempotencyKey を同時に必須化することで、
// 「AllowRetry なのに IdempotencyKey 不在」という不正状態を構築時点で避けます。
func WithRetry(idempotencyKey string) RequestOption {
	return func(r *Request) {
		r.AllowRetry = true
		r.IdempotencyKey = idempotencyKey
	}
}
