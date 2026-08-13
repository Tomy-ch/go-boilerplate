// Package auth は認証に関連する設定を扱うパッケージです。
package auth

import (
	"context"
	"net/http"
	"strings"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/ctxhelper"
	authbd "go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/pkg/xerrors"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/labstack/echo/v5"
)

const prefixBearer = "Bearer "

// NewAuthenticator は、認証用のOpenAPIオプションを返します。
func NewAuthenticator(
	authenticator authbd.Authenticator,
	resolver authbd.IdentityResolver,
) openapi3filter.AuthenticationFunc {
	return func(ctx context.Context, input *openapi3filter.AuthenticationInput) error {
		req := input.RequestValidationInput.Request

		authn, err := authExtractor(ctx, req, authenticator, resolver)
		if err != nil {
			return withHTTPStatus(err)
		}
		if authn == nil {
			return withHTTPStatus(ErrUnauthorizedTokenNotProvided)
		}

		//nolint:contextcheck // input が内包する request の context のスロットへ書き戻すため
		if !ctxhelper.SetAuthn(req.Context(), *authn) {
			return withHTTPStatus(ErrAuthnSlotNotFound)
		}
		return nil
	}
}

// withHTTPStatus は、認証フェーズのエラーへ 401 のステータスを持たせます（元のエラーは保持）。
// OpenAPI バリデータは認証エラーのうち HTTP ステータスを解決できないものを 403 へ丸めるため、
// 401 として返すにはこの段でステータスを持たせる必要があります。
// infra 障害は infraErrorToHTTP が先にステータスを付与するため、ここで 401 に潰されません。
func withHTTPStatus(err error) error {
	var he *echo.HTTPError
	if xerrors.As(err, &he) {
		return err
	}
	return echo.NewHTTPError(http.StatusUnauthorized, "").Wrap(err)
}

// authExtractor は、Bearer トークンを検証して内部ユーザーを解決した Authn を返します。
// トークンが無い場合は nil, nil を返します。
func authExtractor(
	ctx context.Context,
	req *http.Request,
	authenticator authbd.Authenticator,
	resolver authbd.IdentityResolver,
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
	if authn == nil {
		//nolint:nilnil // Authenticate が nil,nil を返した場合は未提供として扱い、呼び出し側で ErrUnauthorizedTokenNotProvided へ変換する
		return nil, nil
	}

	// 認証済みの外部アイデンティティ（issuer + subject）を内部ユーザーへ解決する。
	// 未登録 / 削除済み等の認証失敗（ErrUnauthenticated）は 401となる。
	// DB 等の infra 障害は認証フェーズで 401 に潰さず、apperror の分類に応じた 5xx として返す。
	resolved, err := resolver.Resolve(ctx, authn)
	if err != nil {
		if xerrors.Is(err, apperror.ErrUnauthenticated) {
			return nil, err
		}
		return nil, infraErrorToHTTP(err)
	}
	if resolved == nil {
		// resolver は成功時に非 nil の Authn を返す契約。防御的に未認証として扱う。
		return nil, authbd.ErrIdentityNotFound
	}

	return resolved, nil
}

// infraErrorToHTTP は、 IdentityResolver が返した非認証エラーを、apperror の分類に応じたエラーへ変換します。
func infraErrorToHTTP(err error) error {
	status := http.StatusInternalServerError
	if xerrors.Is(err, apperror.ErrUnavailable) {
		status = http.StatusServiceUnavailable
	}
	return echo.NewHTTPError(status, "").Wrap(err)
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
