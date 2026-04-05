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

		//nolint:contextcheck // inputのContext内部のものにアクセスするため
		ec, ok := ctxhelper.GetEchoContext(req.Context())
		if !ok {
			return ErrUnauthorizedEchoContextNotFound
		}

		authn, err := authExtractor(ctx, req, authCfg, authenticator)
		if err != nil {
			return ErrUnauthorizedInvalidToken
		}
		if authn == nil {
			return ErrUnauthorizedTokenNotProvided
		}

		//nolint:contextcheck // ecのContext内部のものにアクセスするため
		ctxhelper.SetAuthnToEcho(ec, *authn)
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
	token := extractToken(req, authCfg)
	if token == "" {
		return nil, nil
	}

	cred, err := authbd.NewCredential(token)
	if err != nil {
		return nil, err
	}

	authn, err := authenticator.Authenticate(ctx, cred)
	if err != nil {
		return nil, ErrUnauthorizedInvalidToken
	}

	return authn, nil
}

// extractToken は、認証トークンを抽出します。
func extractToken(r *http.Request, authCfg *config.AuthConfig) string {
	// extract from Cookie
	if authCfg.CookieName() != "" {
		if ck, err := r.Cookie(authCfg.CookieName()); err == nil && ck != nil {
			if v := strings.TrimSpace(ck.Value); v != "" {
				return v
			}
		}
	}

	// extract from Header
	if authCfg.HeaderName() == "" {
		return ""
	}

	raw := strings.TrimSpace(r.Header.Get(authCfg.HeaderName()))
	if raw == "" {
		return ""
	}
	if authCfg.AllowedHeaderBearer() && strings.EqualFold(authCfg.HeaderName(), echo.HeaderAuthorization) {
		if strings.HasPrefix(raw, prefixBearer) {
			return strings.TrimSpace(strings.TrimPrefix(raw, prefixBearer))
		}
		return ""
	}
	return raw
}
