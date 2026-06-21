// Package idempotency は、冪等性の入り口 middleware（oapi-codegen StrictMiddleware スロット用）を提供します。
// 役割は opt-in トリガーと ctx 受け渡しのみ。判断・保存は usecase の Run[T] が行います。
package idempotency

import (
	"crypto/sha256"
	"encoding/json"
	"strings"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/ctxhelper"
	idempotencyuc "go-boilerplate/internal/usecase/idempotency"
	"go-boilerplate/pkg/xerrors"

	"github.com/labstack/echo/v4"
)

// headerName は、冪等性キーのリクエストヘッダ名です。
const headerName = "Idempotency-Key"

// maxKeyLength は、Idempotency-Key の最大長です（index 効率 + DoS 防止）。
const maxKeyLength = 255

// NextFunc は、StrictHandlerFunc と構造的に同一の handler 呼び出しシグネチャです。
type NextFunc func(ctx echo.Context, request any) (any, error)

// Middleware は、冪等性入り口を StrictMiddleware の構造的シグネチャで返します。
// 採用する handler パッケージが自身の gen.StrictMiddlewareFunc へ変換して差します。
func Middleware() func(next NextFunc, operationID string) NextFunc {
	return func(next NextFunc, operationID string) NextFunc {
		return func(ec echo.Context, request any) (any, error) {
			return handle(ec, request, operationID, next)
		}
	}
}

func handle(ec echo.Context, request any, operationID string, next NextFunc) (any, error) {
	r := ec.Request()
	key := strings.TrimSpace(r.Header.Get(headerName))
	if key == "" {
		// ヘッダ無し = 非冪等（既存挙動のまま素通し）。
		return next(ec, request)
	}
	if err := validateKey(key); err != nil {
		return nil, err
	}

	// 認証プリンシパルが取れなければ冪等性は発動しない（S1: 認証前提）。
	authn, ok := ctxhelper.GetAuthn(r.Context())
	if !ok {
		return next(ec, request)
	}

	reqCtx := idempotencyuc.WithRequest(r.Context(), idempotencyuc.Request{
		Scope:       authn.Subject(),
		Key:         key,
		Fingerprint: fingerprint(r.Method, r.URL.Path, request),
		Method:      r.Method,
		Path:        r.URL.Path,
		OperationID: operationID,
	})
	ec.SetRequest(r.WithContext(reqCtx))

	return next(ec, request)
}

// validateKey は、Idempotency-Key の健全性を検証します（非空 / ≤255 / 印字可能 ASCII）。違反は 400。
func validateKey(key string) error {
	if len(key) > maxKeyLength {
		return xerrors.Wrap(apperror.ErrInvalidArgument, "Idempotency-Key must be 255 characters or fewer")
	}
	for _, c := range []byte(key) {
		if c < 0x21 || c > 0x7E {
			return xerrors.Wrap(apperror.ErrInvalidArgument, "Idempotency-Key must contain only printable ASCII characters")
		}
	}
	return nil
}

// fingerprint は、method + path + typed request の決定的 marshal を SHA-256 した指紋を返します。
func fingerprint(method, path string, request any) []byte {
	h := sha256.New()
	h.Write([]byte(method))
	h.Write([]byte{'\n'})
	h.Write([]byte(path))
	h.Write([]byte{'\n'})
	if b, err := json.Marshal(request); err == nil {
		h.Write(b)
	}
	return h.Sum(nil)
}
