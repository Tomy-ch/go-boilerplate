//go:generate oapi-codegen --include-tags=v1/prefectures --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/prefectures --package=gen --generate=echo5-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package prefectures は、/v1/prefectures エンドポイントに関連するハンドラを提供します。
package prefectures

import (
	"context"

	"go-boilerplate/internal/controller/handler/v1/prefectures/gen"
	"go-boilerplate/internal/observability"
	prefectureuc "go-boilerplate/internal/usecase/prefecture"

	"github.com/labstack/echo/v4"
)

type server struct {
	tracer observability.LayerTracer
	uc     prefectureuc.Usecase
}

// BindHandler は、都道府県マスタ一覧のハンドラを Echo に登録します。
func BindHandler(e *echo.Echo, tf observability.TracerFactory, uc prefectureuc.Usecase) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{
		tracer: tf.Controller(),
		uc:     uc,
	}, nil))
}

// GetPrefectures は、都道府県マスタの全件を code 昇順で返します。
func (s *server) GetPrefectures(
	ctx context.Context, _ gen.GetPrefecturesRequestObject,
) (gen.GetPrefecturesResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	list, err := s.uc.ListPrefectures(ctx)
	if err != nil {
		return nil, err
	}

	prefectures := make([]gen.PrefectureResponse, len(list))
	for i, dto := range list {
		prefectures[i] = toPrefectureResponse(dto)
	}

	return gen.GetPrefectures200JSONResponse(prefectures), nil
}

// toPrefectureResponse は、ユースケースの DTO を HTTP レスポンスへ変換します。
func toPrefectureResponse(dto prefectureuc.PrefectureDTO) gen.PrefectureResponse {
	return gen.PrefectureResponse{
		Id:   dto.ID.ToPrimitive(),
		Code: dto.Code,
		Name: dto.Name,
	}
}
