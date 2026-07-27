//go:generate oapi-codegen --include-tags=v1/addresses --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/addresses --package=gen --generate=echo5-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package addresses は、/v1/addresses エンドポイントに関連するハンドラを提供します。
package addresses

import (
	"context"
	"strings"

	"go-boilerplate/internal/controller/handler/v1/addresses/gen"
	"go-boilerplate/internal/observability"
	addressuc "go-boilerplate/internal/usecase/address"
	"go-boilerplate/pkg/uuid"

	"github.com/labstack/echo/v4"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

type server struct {
	tracer observability.LayerTracer
	uc     addressuc.Usecase
}

// BindHandler は、郵便番号住所補完エンドポイントのハンドラを Echo に登録します。
func BindHandler(e *echo.Echo, tf observability.TracerFactory, uc addressuc.Usecase) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{
		tracer: tf.Controller(),
		uc:     uc,
	}, nil))
}

// GetAddresses は、郵便番号から住所候補を補完して返します。
func (s *server) GetAddresses(
	ctx context.Context, request gen.GetAddressesRequestObject,
) (gen.GetAddressesResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	// リクエスト形の正規化は controller の責務。ハイフンを除去し、内層へは 7 桁の数字列を渡す。
	postalCode := strings.ReplaceAll(request.Params.PostalCode, "-", "")

	result, err := s.uc.LookupByPostalCode(ctx, postalCode)
	if err != nil {
		return nil, err
	}

	return gen.GetAddresses200JSONResponse(gen.AddressCandidatesResponse{
		Candidates: toAddressCandidates(result.Candidates),
		IsFallback: result.IsFallback,
	}), nil
}

// toAddressCandidates は、usecase の住所候補一覧を API レスポンス DTO へ変換します。
func toAddressCandidates(views []*addressuc.CandidateView) []gen.AddressCandidate {
	candidates := make([]gen.AddressCandidate, len(views))
	for i, v := range views {
		candidates[i] = gen.AddressCandidate{
			PrefectureId:   toNullableUUID(v.PrefectureID),
			PrefectureName: v.PrefectureName,
			City:           v.City,
			Town:           v.Town,
		}
	}
	return candidates
}

// toNullableUUID は、県名解決不能時に nil となる prefecture_id を nullable なレスポンス型へ変換します。
func toNullableUUID(id *uuid.UUID) *openapi_types.UUID {
	if id == nil {
		return nil
	}
	primitive := id.ToPrimitive()
	return &primitive
}
