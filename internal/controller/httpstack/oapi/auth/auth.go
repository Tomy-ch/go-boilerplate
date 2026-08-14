// Package auth は認証に関連する設定を扱うパッケージです。
package auth

import (
	"context"
	"net/http"
	"strings"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/error/response"
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
	return func(_ context.Context, input *openapi3filter.AuthenticationInput) error {
		req := input.RequestValidationInput.Request

		// OpenAPI バリデータが渡す context は context.Background() から組み立てられており、
		// スパン・deadline・キャンセルのいずれも持たない。認証は request の予算の内側で行う。
		//nolint:contextcheck // 引数の context ではなく input が内包する request の context を用いるため
		authn, err := authExtractor(req.Context(), req, authenticator, resolver)
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

// withHTTPStatus は、認証フェーズのエラーへ apperror の分類に対応するステータスを持たせます（元のエラーは保持）。
// OpenAPI バリデータはステータスを解決できないエラーを 403 へ丸めるため、この段で持たせないと、
// 認可の判定を行っていないものが認可の結論として外へ出ます。
// メッセージは空のまま返します（本文は errorhandler が組み立てる）。
func withHTTPStatus(err error) error {
	var he *echo.HTTPError
	if xerrors.As(err, &he) {
		return err
	}
	return echo.NewHTTPError(response.NewHTTPErrorFromAppError(err).HTTPStatus, "").Wrap(err)
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

	// 資格情報についての結論だけを認証エラーへ寄せ、原因は外へ出さない。
	// 検証を遂行できなかった場合（鍵が引けない / 予算切れ）は結論ではないため、分類を保ったまま返す。
	authn, err := authenticator.Authenticate(ctx, cred)
	if err != nil {
		if xerrors.Is(err, apperror.ErrUnauthenticated) {
			return nil, ErrUnauthorizedInvalidToken
		}
		return nil, err
	}
	if authn == nil {
		//nolint:nilnil // Authenticate が nil,nil を返した場合は未提供として扱い、呼び出し側で ErrUnauthorizedTokenNotProvided へ変換する
		return nil, nil
	}

	// 認証済みの外部アイデンティティ（issuer + subject）を内部ユーザーへ解決する。エラーは分類を変えずに返す。
	resolved, err := resolver.Resolve(ctx, authn)
	if err != nil {
		return nil, err
	}
	if resolved == nil {
		// resolver は成功時に非 nil の Authn を返す契約。防御的に未認証として扱う。
		return nil, authbd.ErrIdentityNotFound
	}

	return resolved, nil
}

// extractBearerToken は、Authorization ヘッダから Bearer トークンを抽出します（ヘッダ名・スキームは RFC 6750 により固定）。
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
