// Package auth は、query の stream ticket を検証する securityScheme（StreamTicket）の認証器を提供します。
// 検証そのものは usecase/realtime の TicketVerifier が行い、ここは spec の宣言から生値と destination を取り出して
// 結果を StreamGrant スロットへ書くだけです。
package auth

import (
	"context"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/ctxhelper"
	oapiauth "go-boilerplate/internal/controller/httpstack/oapi/auth"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	ucrealtime "go-boilerplate/internal/usecase/realtime"
	"go-boilerplate/pkg/xerrors"

	"github.com/getkin/kin-openapi/openapi3filter"
)

const (
	// SchemeName は、担当する securityScheme の名前（openapi/openapi.yaml の securitySchemes のキー）です。
	SchemeName = "StreamTicket"
	// destinationParam は、接続先 stream を運ぶ path パラメータの名前です。
	destinationParam = "destination"
)

var (
	// ErrSchemeDeclarationMissing は、validator が scheme の宣言を渡さず、ticket を運ぶパラメータ名が分からない場合のエラー。
	ErrSchemeDeclarationMissing = xerrors.Wrap(apperror.ErrInternal, "stream ticket scheme declaration is missing")
	// ErrStreamGrantSlotNotFound は、StreamGrant スロット未装着の場合のエラー（資格情報と無関係な結線の不具合）。
	ErrStreamGrantSlotNotFound = xerrors.Wrap(apperror.ErrInternal, "stream grant slot not found in request context")
)

type streamTicket struct {
	verifier ucrealtime.TicketVerifier
}

// New は、StreamTicket scheme の認証器を返します。
func New(verifier ucrealtime.TicketVerifier) oapiauth.SchemeAuthenticator {
	return &streamTicket{verifier: verifier}
}

// Scheme は、SchemeName を返します。
func (s *streamTicket) Scheme() string {
	return SchemeName
}

// Authenticate は、scheme が宣言した query パラメータの ticket を path の destination に対して検証し、
// 通れば束縛を StreamGrant スロットへ書き込みます。ticket の生値はエラーにも context にも残しません
// — 後段が受け取るのは、同じ ticket をもう一度検証する関数値だけです。
func (s *streamTicket) Authenticate(_ context.Context, input *openapi3filter.AuthenticationInput) error {
	if input.SecurityScheme == nil || input.SecurityScheme.Name == "" {
		return ErrSchemeDeclarationMissing
	}

	in := input.RequestValidationInput
	// OpenAPI バリデータが渡す context は context.Background() から組み立てられており、
	// スパン・deadline・キャンセルのいずれも持たない。検証は request の予算の内側で行う。
	ctx := in.Request.Context()

	value := in.GetQueryParams().Get(input.SecurityScheme.Name)
	destination := rt.StreamID(in.PathParams[destinationParam])

	//nolint:contextcheck // 引数の context ではなく input が内包する request の context を用いるため
	grant, err := s.verifier.Verify(ctx, value, destination)
	if err != nil {
		return err
	}

	//nolint:contextcheck // input が内包する request の context のスロットへ書き戻すため
	if !ctxhelper.SetStreamGrant(ctx, grant) {
		return ErrStreamGrantSlotNotFound
	}

	//nolint:contextcheck // 同上
	if !ctxhelper.SetStreamRevalidator(ctx, func(ctx context.Context) error {
		_, err := s.verifier.Verify(ctx, value, destination)

		return err
	}) {
		return ErrStreamGrantSlotNotFound
	}

	return nil
}
