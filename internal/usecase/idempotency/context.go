// Package idempotency は、冪等性の at-most-once 実行と replay を行う Run[T] を提供します。
package idempotency

import "context"

type requestCtxKey struct{}

// Request は、冪等性判定に必要な HTTP 非依存のリクエスト情報です。
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
