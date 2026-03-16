//go:generate oapi-codegen --include-tags=v1/users --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/users --package=gen --generate=echo-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package users は、/v1/users エンドポイントに関連するハンドラを提供します。
package users

import (
	"context"

	"boilerplate-go/internal/apperror"
	"boilerplate-go/internal/controller/ctxhelper"
	"boilerplate-go/internal/controller/handler/v1/users/gen"
	"boilerplate-go/internal/observability"
	"boilerplate-go/internal/usecase/tools/paging"
	"boilerplate-go/internal/usecase/user"
	"boilerplate-go/pkg/ptr"
	"boilerplate-go/pkg/xerrors"

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

	params := &user.GetParamsDTO{
		Keyword: request.Params.Keyword,
		Active:  request.Params.Active,
	}

	dtos, err := s.uc.ListUsersByKeyword(ctx, params, page)
	if err != nil {
		return nil, err
	}

	users := make([]gen.UserResponse, len(dtos))
	for i, dto := range dtos {
		users[i] = gen.UserResponse{
			FirstName:  dto.FirstName,
			LastName:   dto.LastName,
			Email:      types.Email(dto.Email),
			Phone:      ptr.To(dto.Phone),
			PostalCode: dto.PostalCode,
			Prefecture: dto.PrefectureName,
			City:       dto.City,
			Street:     dto.Street,
			Building:   dto.Building,
		}
	}

	res := gen.UsersResponse{
		Users:  users,
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

	createPrams := &user.CreateParamsDTO{}
	createPrams.UserID = userID
	createPrams.FirstName = request.Body.FirstName
	createPrams.LastName = request.Body.LastName
	createPrams.Email = string(request.Body.Email)
	createPrams.Phone = request.Body.Phone
	createPrams.PostalCode = request.Body.PostalCode
	createPrams.PrefectureName = request.Body.Prefecture
	createPrams.City = request.Body.City
	createPrams.Street = request.Body.Street
	createPrams.Building = request.Body.Building
	createPrams.RawPassword = request.Body.Password

	dto, err := s.uc.CreateUser(ctx, createPrams)
	if err != nil {
		return nil, err
	}

	res := gen.UserResponse{
		FirstName:  dto.FirstName,
		LastName:   dto.LastName,
		Email:      types.Email(dto.Email),
		Phone:      ptr.To(dto.Phone),
		PostalCode: dto.PostalCode,
		Prefecture: dto.PrefectureName,
		City:       dto.City,
		Street:     dto.Street,
		Building:   dto.Building,
	}

	return gen.PostUsers201JSONResponse(res), nil
}
