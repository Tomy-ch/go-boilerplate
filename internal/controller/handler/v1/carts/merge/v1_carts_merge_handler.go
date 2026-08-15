//go:generate oapi-codegen --include-tags=v1/carts/merge --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/carts/merge --package=gen --generate=echo5-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package merge は、/v1/carts/me/merge エンドポイントに関連するハンドラを提供します。
package merge

import (
	"context"
	"net/http"

	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/v1/carts/merge/gen"
	idempotencymw "go-boilerplate/internal/controller/httpstack/idempotency"
	"go-boilerplate/internal/observability"
	cartuc "go-boilerplate/internal/usecase/cart"
	"go-boilerplate/internal/usecase/idempotency"
	"go-boilerplate/pkg/uuid"

	"github.com/labstack/echo/v5"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

type server struct {
	tracer observability.LayerTracer
	uc     cartuc.Usecase
	idem   idempotency.Deps
}

// BindHandler は、カート引き継ぎのハンドラを Echo に登録します。冪等ミドルウェアを併用します。
func BindHandler(
	e *echo.Echo,
	tf observability.TracerFactory,
	uc cartuc.Usecase,
	idem idempotency.Deps,
) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{
		tracer: tf.Controller(),
		uc:     uc,
		idem:   idem,
	}, []gen.StrictMiddlewareFunc{idempotencymw.StrictMiddleware[gen.StrictHandlerFunc]()}))
}

// PostCartsMeMerge は、ゲストカートを呼び出し主体へ引き継ぎます。
func (s *server) PostCartsMeMerge(
	ctx context.Context, request gen.PostCartsMeMergeRequestObject,
) (gen.PostCartsMeMergeResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	userID, err := ctxhelper.RequireUserID(ctx)
	if err != nil {
		return nil, err
	}

	result, _, err := idempotency.Run(ctx, s.idem, http.StatusOK,
		func(ctx context.Context) (cartuc.MergeCartResult, error) {
			return s.uc.MergeOnLogin(ctx, cartuc.MergeOnLoginParams{
				UserID:       userID,
				SessionToken: request.Params.XCartSession,
			})
		})
	if err != nil {
		return nil, err
	}

	return gen.PostCartsMeMerge200JSONResponse(toCartMergeResponse(result)), nil
}

// toCartMergeResponse は、ユースケースの DTO を HTTP レスポンスへ変換します。
func toCartMergeResponse(result cartuc.MergeCartResult) gen.CartMergeResponse {
	return gen.CartMergeResponse{
		Clamped: toUUIDs(result.Clamped),
		Dropped: toUUIDs(result.Dropped),
	}
}

// toUUIDs は、報告に載せる商品 ID を応答の型へ変換します。
// 空の場合も nil ではなく空スライスを返します（応答で null と [] を混ぜないため）。
func toUUIDs(ids []uuid.UUID) []openapi_types.UUID {
	converted := make([]openapi_types.UUID, len(ids))
	for i, id := range ids {
		converted[i] = id.ToPrimitive()
	}
	return converted
}
