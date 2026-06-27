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

import (
	"context"
	"slices"
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
// `Method("garbage")` のような任意の文字列からの生成はコンパイル時に排除されます
// （string 別名で型が防げない問題を根絶）。ゼロ値 Method{} のみ構築可能で、これは Do が
// ErrInvalidArgument で弾きます。
type Method struct{ s string }

// Header は、HTTP ヘッダを表す自前型です（net/http.Header を露出しません）。
type Header map[string][]string

// Request は、1 回の外部 HTTP 呼び出しの意図を表します。
//
// 構築は NewRequest（必須項目をシグネチャで強制）と With* オプション経由のみ。任意メソッド文字列は
// 型でコンパイル時に排除し、「空メソッド」「AllowRetry なのに IdempotencyKey 不在」といった残りの
// 不正状態は Do 実行時に ErrInvalidArgument で弾きます。
type Request struct {
	// downstream は、論理依存名です（必須）。breaker / metrics / profile / budget のキーになります。
	downstream Downstream
	// method は、HTTP メソッドです。
	method Method
	// url は、リクエスト先 URL です。
	// 認証情報などの機密はクエリではなく Header に載せてください（URL は o11y span へ記録されます）。
	url string
	// header は、リクエストヘッダです。
	header Header
	// body は、バッファ済みのリクエストボディです。retry で replay 可能です（stream は非対象）。
	body []byte
	// idempotencyKey は、非冪等メソッドを retry 安全にするためのキーです（任意）。
	idempotencyKey string
	// allowRetry は、POST 等の非冪等メソッドを明示的に retry 許可するフラグです。
	// true の場合は idempotencyKey が必須です（二重実行防止）。
	allowRetry bool
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

// MethodGet は、HTTP GET メソッドを返します。
//
// パッケージ外からの再代入で定義済み値が破壊されるのを防ぐため、関数として提供します。
func MethodGet() Method { return Method{"GET"} }

// MethodPost は、HTTP POST メソッドを返します。
func MethodPost() Method { return Method{"POST"} }

// MethodPut は、HTTP PUT メソッドを返します。
func MethodPut() Method { return Method{"PUT"} }

// MethodPatch は、HTTP PATCH メソッドを返します。
func MethodPatch() Method { return Method{"PATCH"} }

// MethodDelete は、HTTP DELETE メソッドを返します。
func MethodDelete() Method { return Method{"DELETE"} }

// String は、HTTP メソッド文字列を返します（ゼロ値は ""）。
func (m Method) String() string { return m.s }

// NewRequest は、必須項目（method / downstream / url）をシグネチャで強制してリクエストを生成します。
// 任意メソッド文字列は型で排除済み。空メソッドや AllowRetry の key 不在は Do 実行時に弾かれます。
func NewRequest(method Method, downstream Downstream, url string, opts ...RequestOption) *Request {
	r := &Request{
		downstream: downstream,
		method:     method,
		url:        url,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// WithHeader は、リクエストヘッダを設定します。
func WithHeader(h Header) RequestOption { return func(r *Request) { r.header = h } }

// WithBody は、リクエストボディを設定します。
func WithBody(b []byte) RequestOption { return func(r *Request) { r.body = b } }

// WithIdempotencyKey は、冪等性キーを設定します（retry は許可しません）。
func WithIdempotencyKey(key string) RequestOption {
	return func(r *Request) { r.idempotencyKey = key }
}

// WithRetry は、非冪等メソッドの retry を許可し idempotencyKey を同時に設定します。
// 空 key のまま（WithRetry("")）Do を呼ぶと ErrInvalidArgument で弾かれます（二重実行防止）。
func WithRetry(idempotencyKey string) RequestOption {
	return func(r *Request) {
		r.allowRetry = true
		r.idempotencyKey = idempotencyKey
	}
}

// Downstream は、論理依存名を返します。
func (r *Request) Downstream() Downstream { return r.downstream }

// Method は、HTTP メソッドを返します。
func (r *Request) Method() Method { return r.method }

// URL は、リクエスト先 URL を返します。
func (r *Request) URL() string { return r.url }

// Header は、リクエストヘッダの防御的コピーを返します（Request は不変値オブジェクトのため内部状態を共有しません）。
func (r *Request) Header() Header { return r.header.clone() }

// Body は、リクエストボディの防御的コピーを返します（Request は不変値オブジェクトのため内部状態を共有しません）。
func (r *Request) Body() []byte { return slices.Clone(r.body) }

// IdempotencyKey は、冪等性キーを返します（未設定時は ""）。
func (r *Request) IdempotencyKey() string { return r.idempotencyKey }

// AllowRetry は、非冪等メソッドの retry 許可フラグを返します。
func (r *Request) AllowRetry() bool { return r.allowRetry }

// clone は、Header の深いコピーを返します（map と各値スライスを複製し、内部状態との共有を断ちます）。
func (h Header) clone() Header {
	if h == nil {
		return nil
	}
	out := make(Header, len(h))
	for k, v := range h {
		out[k] = slices.Clone(v)
	}
	return out
}
