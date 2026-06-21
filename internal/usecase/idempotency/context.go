// Package idempotency は、冪等性管理の本体（Run[T]）を提供します。
// 入り口 middleware（controller）が Request を ctx へ stash し、Run がそれを読んで
// 業務 tx 内で at-most-once 実行 / replay / 409 / 422 を判定します。
package idempotency

import "context"

type requestCtxKey struct{}

// Request は、入り口 middleware が ctx へ載せる冪等性情報です。
// すべて HTTP 非依存の素の値で、usecase の Run が読みます。
type Request struct {
	// Scope は、認証プリンシパル ID（越境防止の名前空間）。
	Scope string
	// Key は、クライアント供給の Idempotency-Key。
	Key string
	// Fingerprint は、リクエスト指紋（SHA-256・32 byte）。
	Fingerprint []byte
	// Method / Path は、リクエストの method / path。
	Method string
	Path   string
	// OperationID は、o11y ラベル用（OpenAPI operationId）。
	OperationID string
}

// WithRequest は、冪等性 Request を ctx へ載せます（入り口 middleware が使用）。
func WithRequest(ctx context.Context, r Request) context.Context {
	return context.WithValue(ctx, requestCtxKey{}, r)
}

// requestFromContext は、ctx から冪等性 Request を取り出します（無ければ ok=false＝非冪等）。
func requestFromContext(ctx context.Context) (Request, bool) {
	r, ok := ctx.Value(requestCtxKey{}).(Request)
	return r, ok
}
