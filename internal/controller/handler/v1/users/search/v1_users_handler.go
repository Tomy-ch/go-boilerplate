//go:generate oapi-codegen --include-tags=v1/users/search --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/users/search --package=gen --generate=echo-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package search は、/v1/users/search エンドポイントに関連するハンドラを提供します。
package search

import (
	"context"

	"go-boilerplate/internal/controller/handler/v1/users/search/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/tools/paging"
	search "go-boilerplate/internal/usecase/user/search"

	"github.com/labstack/echo/v4"
	"github.com/oapi-codegen/runtime/types"
)

type server struct {
	tracer observability.LayerTracer
	uc     search.Usecase
}

// BindHandler は、ユーザー検索エンドポイントのハンドラーをEchoに登録します。
func BindHandler(e *echo.Echo, tf observability.TracerFactory, us search.Usecase) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{
		tracer: tf.Controller(),
		uc:     us,
	}, nil))
}

// GetUsersSearch は、検索条件に基づいてユーザーの一覧を取得します。
func (s *server) GetUsersSearch(ctx context.Context, request gen.GetUsersSearchRequestObject) (gen.GetUsersSearchResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	page, err := paging.NewPagingFrom1Based(request.Params.Page, request.Params.PerPage)
	if err != nil {
		return nil, err
	}

	filter := &search.SearchParams{
		Keyword: request.Params.Keyword,
		Active:  request.Params.Active,
	}
	list, err := s.uc.ListUsersByKeywordWithTotal(ctx, filter, page)
	if err != nil {
		return nil, err
	}

	users := make([]gen.UsersSearchResponseItem, len(list.Items))
	for i, dto := range list.Items {
		users[i] = gen.UsersSearchResponseItem{
			FirstName:    dto.FirstName,
			LastName:     dto.LastName,
			Email:        types.Email(dto.Email),
			Phone:        dto.Phone,
			PostalCode:   dto.PostalCode,
			Prefecture:   dto.PrefectureName,
			City:         dto.City,
			Street:       dto.Street,
			Building:     dto.Building,
			RegisteredAt: dto.RegisteredAt,
			DeletedAt:    dto.DeletedAt,
		}
	}

	return gen.GetUsersSearch200JSONResponse(gen.UsersSearchResponse{
		Users:  users,
		Total:  list.Total,
		Limit:  page.Limit(),
		Offset: page.Offset(),
	}), nil
}
