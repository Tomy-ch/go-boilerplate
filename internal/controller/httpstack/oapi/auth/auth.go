// Package auth は認証に関連する設定を扱うパッケージです。
package auth

import (
	"context"
	"net/http"
	"strings"

	"go-boilerplate/internal/controller/ctxhelper"
	authbd "go-boilerplate/internal/usecase/boundary/auth"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/labstack/echo/v4"
)

const prefixBearer = "Bearer "

// NewAuthenticator は、認証用のOpenAPIオプションを返します。
func NewAuthenticator(
	authenticator authbd.Authenticator,
) openapi3filter.AuthenticationFunc {
	return func(ctx context.Context, input *openapi3filter.AuthenticationInput) error {
		req := input.RequestValidationInput.Request

		// エラー変換は authExtractor に一本化。
		authn, err := authExtractor(ctx, req, authenticator)
		if err != nil {
			return err
		}
		if authn == nil {
			return ErrUnauthorizedTokenNotProvided
		}

		//nolint:contextcheck // input が内包する request の context のスロットへ書き戻すため
		if !ctxhelper.SetAuthn(req.Context(), *authn) {
			return ErrAuthnSlotNotFound
		}
		return nil
	}
}

// authExtractor は、認証情報を抽出します。
func authExtractor(
	ctx context.Context,
	req *http.Request,
	authenticator authbd.Authenticator,
) (*authbd.Authn, error) {
	scheme, token := extractBearerToken(req)
	if token == "" {
		//nolint:nilnil // トークン未提供を表す。呼び出し側で authn==nil を判定し未提供エラーへ変換するため意図的にnil,nilを返す
		return nil, nil
	}

	cred, err := authbd.NewCredential(scheme, token)
	if err != nil {
		return nil, err
	}

	authn, err := authenticator.Authenticate(ctx, cred)
	if err != nil {
		return nil, ErrUnauthorizedInvalidToken
	}

	return authn, nil
}

// extractBearerToken は、Authorization ヘッダから Bearer トークンを抽出します。
// Bearer トークンは RFC 6750 で Authorization ヘッダに固定されるため、ヘッダ名・スキームは可変にしない。
// Authorization: Bearer <token> 形式のときだけ scheme=SchemeBearer と token を返し、
// それ以外は scheme/token とも空を返します。
func extractBearerToken(r *http.Request) (string, string) {
	raw := strings.TrimSpace(r.Header.Get(echo.HeaderAuthorization))
	if raw == "" {
		return "", ""
	}
	after, ok := strings.CutPrefix(raw, prefixBearer)
	if !ok {
		return "", ""
	}
	return authbd.SchemeBearer, strings.TrimSpace(after)
}
