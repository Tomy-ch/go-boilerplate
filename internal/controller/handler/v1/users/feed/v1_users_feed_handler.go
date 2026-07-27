//go:generate oapi-codegen --include-tags=v1/users/feed --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/users/feed --package=gen --generate=echo5-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package feed は、/v1/users/feed エンドポイントに関連するハンドラを提供します。
package feed

import (
	"context"

	"go-boilerplate/internal/controller/handler/v1/users/feed/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/tools/paging"
	"go-boilerplate/internal/usecase/user"

	"github.com/labstack/echo/v4"
	"github.com/oapi-codegen/runtime/types"
)

type server struct {
	tracer observability.LayerTracer
	uc     user.Usecase
}

// BindHandler は、ユーザーフィードのハンドラを Echo に登録します。
func BindHandler(e *echo.Echo, tf observability.TracerFactory, uc user.Usecase) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{
		tracer: tf.Controller(),
		uc:     uc,
	}, nil))
}

// GetUsersFeed は、未削除ユーザーを作成日時の降順（cursor ページネーション）で取得します。
func (s *server) GetUsersFeed(ctx context.Context, request gen.GetUsersFeedRequestObject) (gen.GetUsersFeedResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	// WARN: このエンドポイントは認可チェックを実施していません。
	cursor, err := paging.NewCursor(request.Params.After, request.Params.First)
	if err != nil {
		return nil, err
	}

	feed, err := s.uc.ListUsersFeed(ctx, cursor)
	if err != nil {
		return nil, err
	}

	users := make([]gen.UserResponse, len(feed.Items))
	for i, dto := range feed.Items {
		users[i] = toUserResponse(dto)
	}

	res := gen.UsersFeedResponse{
		Users:      users,
		NextCursor: feed.NextCursor,
		HasNext:    feed.NextCursor != nil,
	}

	return gen.GetUsersFeed200JSONResponse(res), nil
}

// toUserResponse は、ユースケースのDTOをHTTPレスポンスへ変換します。
func toUserResponse(dto user.UserView) gen.UserResponse {
	return gen.UserResponse{
		FirstName:  dto.FirstName,
		LastName:   dto.LastName,
		Email:      types.Email(dto.Email),
		Phone:      dto.Phone,
		PostalCode: dto.PostalCode,
		Prefecture: dto.PrefectureName,
		City:       dto.City,
		Street:     dto.Street,
		Building:   dto.Building,
		DeletedAt:  dto.DeletedAt,
	}
}
