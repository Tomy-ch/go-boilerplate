// Package auth は認証に関連する設定を扱うパッケージです。
package auth

import (
	"context"
	"net/http"
	"strings"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/ctxhelper"
	authbd "go-boilerplate/internal/usecase/boundary/auth"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/labstack/echo/v4"
)

const prefixBearer = "Bearer "

// NewAuthenticator は、認証用のOpenAPIオプションを返します。
func NewAuthenticator(
	authCfg *config.AuthConfig,
	authenticator authbd.Authenticator,
) openapi3filter.AuthenticationFunc {
	return func(ctx context.Context, input *openapi3filter.AuthenticationInput) error {
		req := input.RequestValidationInput.Request

		// エラー変換は authExtractor に一本化。
		authn, err := authExtractor(ctx, req, authCfg, authenticator)
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
	authCfg *config.AuthConfig,
	authenticator authbd.Authenticator,
) (*authbd.Authn, error) {
	scheme, token := extractToken(req, authCfg)
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

// extractToken は、authCfg.HeaderName() で設定された認証ヘッダーからスキームとトークンを抽出します。
// ヘッダーが Authorization かつ Bearer 許可時のみ Bearer プレフィックスを検証して scheme に SchemeBearer を設定し、
// それ以外はヘッダー値をそのまま token として返し scheme は空文字になります。
func extractToken(r *http.Request, authCfg *config.AuthConfig) (string, string) {
	if authCfg.HeaderName() == "" {
		return "", ""
	}

	raw := strings.TrimSpace(r.Header.Get(authCfg.HeaderName()))
	if raw == "" {
		return "", ""
	}
	if authCfg.AllowedHeaderBearer() && strings.EqualFold(authCfg.HeaderName(), echo.HeaderAuthorization) {
		if after, ok := strings.CutPrefix(raw, prefixBearer); ok {
			return authbd.SchemeBearer, strings.TrimSpace(after)
		}
		return "", ""
	}
	return "", raw
}
