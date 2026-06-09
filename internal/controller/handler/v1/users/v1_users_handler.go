//go:generate oapi-codegen --include-tags=v1/users --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/users --package=gen --generate=echo-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package users は、/v1/users エンドポイントに関連するハンドラを提供します。
package users

import (
	"context"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/v1/users/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/tools/paging"
	"go-boilerplate/internal/usecase/user"
	"go-boilerplate/pkg/xerrors"

	"github.com/labstack/echo/v4"
	"github.com/oapi-codegen/runtime/types"
)

var ErrUnauthenticatedUser = xerrors.Wrap(apperror.ErrUnauthenticated, "requires authenticated user")

type server struct {
	tracer observability.LayerTracer
	uc     user.Usecase
}

func BindHandler(e *echo.Echo, tf observability.TracerFactory, uc user.Usecase) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{
		tracer: tf.Controller(),
		uc:     uc,
	}, nil))
}

// GetUsers は、ユーザー一覧を取得します。
func (s *server) GetUsers(ctx context.Context, request gen.GetUsersRequestObject) (gen.GetUsersResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	// WARN: 本来はここで認可を行うべきですが、今回は省略します。
	page, err := paging.NewPagingFrom1Based(request.Params.Page, request.Params.PerPage)
	if err != nil {
		return nil, err
	}

	dtos, err := s.uc.ListUsers(ctx, request.Params.Active, page)
	if err != nil {
		return nil, err
	}

	total, err := s.uc.CountUsers(ctx, request.Params.Active)
	if err != nil {
		return nil, err
	}

	users := make([]gen.UserResponse, len(dtos))
	for i, dto := range dtos {
		users[i] = toUserResponse(dto)
	}

	res := gen.UsersResponse{
		Users:  users,
		Total:  total,
		Limit:  page.Limit(),
		Offset: page.Offset(),
	}

	return gen.GetUsers200JSONResponse(res), nil
}

// PostUsers は、ユーザーを作成します。
func (s *server) PostUsers(ctx context.Context, request gen.PostUsersRequestObject) (gen.PostUsersResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	authn, ok := ctxhelper.GetAuthn(ctx)
	if !ok {
		return nil, ErrUnauthenticatedUser
	}
	userID, err := authn.ID()
	if err != nil {
		return nil, xerrors.Wrap(err, "failed to get user ID from authenticator")
	}

	createParams := &user.CreateParamsDTO{
		UserID:      userID,
		RawPassword: request.Body.Password,
		MutableFields: user.MutableFields{
			FirstName:      request.Body.FirstName,
			LastName:       request.Body.LastName,
			Email:          string(request.Body.Email),
			Phone:          request.Body.Phone,
			PostalCode:     request.Body.PostalCode,
			PrefectureName: request.Body.Prefecture,
			City:           request.Body.City,
			Street:         request.Body.Street,
			Building:       request.Body.Building,
		},
	}

	dto, err := s.uc.CreateUser(ctx, createParams)
	if err != nil {
		return nil, err
	}

	return gen.PostUsers201JSONResponse(toUserResponse(dto)), nil
}

// toUserResponse は、ユースケースのDTOをHTTPレスポンスへ変換します。
func toUserResponse(dto user.MutableFields) gen.UserResponse {
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
