//go:generate oapi-codegen --include-tags=v1/users/detail --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/users/detail --package=gen --generate=echo5-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package detail は、/v1/users/{userId} エンドポイントに関連するハンドラを提供します。
package detail

import (
	"context"

	"go-boilerplate/internal/controller/conv"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/v1/users/detail/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/user"
	"go-boilerplate/pkg/xerrors"

	"github.com/labstack/echo/v5"
	"github.com/oapi-codegen/runtime/types"
)

type server struct {
	tracer observability.LayerTracer
	uc     user.Usecase
}

// BindHandler は、ユーザー詳細のハンドラを Echo に登録します。
func BindHandler(e *echo.Echo, tf observability.TracerFactory, uc user.Usecase) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{
		tracer: tf.Controller(),
		uc:     uc,
	}, nil))
}

// GetUsersDetail は、指定されたUUIDに該当するユーザーの詳細情報を取得します。
func (s *server) GetUsersDetail(ctx context.Context, request gen.GetUsersDetailRequestObject) (gen.GetUsersDetailResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	authn, err := ctxhelper.RequireAuthn(ctx)
	if err != nil {
		return nil, err
	}

	id := conv.UUID(request.UserId)

	dto, err := s.uc.GetUser(ctx, &authn, id)
	if err != nil {
		return nil, err
	}

	return gen.GetUsersDetail200JSONResponse(toUserResponse(dto)), nil
}

// GetUsersMe は、認証コンテキストの内部 UserID に該当するユーザー自身の詳細情報を取得します。
func (s *server) GetUsersMe(ctx context.Context, _ gen.GetUsersMeRequestObject) (gen.GetUsersMeResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	authn, err := ctxhelper.RequireAuthn(ctx)
	if err != nil {
		return nil, err
	}
	id, err := authn.UserID()
	if err != nil {
		return nil, xerrors.Wrap(err, "failed to get user ID from authenticator")
	}

	dto, err := s.uc.GetUser(ctx, &authn, id)
	if err != nil {
		return nil, err
	}

	return gen.GetUsersMe200JSONResponse(toUserResponse(dto)), nil
}

// PutUsersDetail は、指定されたUUIDに該当するユーザーのプロフィールを全て更新します。
func (s *server) PutUsersDetail(ctx context.Context, request gen.PutUsersDetailRequestObject) (gen.PutUsersDetailResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	authn, err := ctxhelper.RequireAuthn(ctx)
	if err != nil {
		return nil, err
	}

	id := conv.UUID(request.UserId)

	dto := &user.UpdateProfileParams{
		FirstName:      request.Body.FirstName,
		LastName:       request.Body.LastName,
		Email:          conv.Email(request.Body.Email),
		Phone:          request.Body.Phone,
		PostalCode:     request.Body.PostalCode,
		PrefectureName: request.Body.Prefecture,
		City:           request.Body.City,
		Street:         request.Body.Street,
		Building:       request.Body.Building,
	}

	res, err := s.uc.UpdateUser(ctx, &authn, id, dto)
	if err != nil {
		return nil, err
	}

	return gen.PutUsersDetail200JSONResponse(toUserResponse(res)), nil
}

// PatchUsersDetail は、指定されたUUIDに該当するユーザーの情報を部分的に更新します。
func (s *server) PatchUsersDetail(ctx context.Context, request gen.PatchUsersDetailRequestObject) (gen.PatchUsersDetailResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	authn, err := ctxhelper.RequireAuthn(ctx)
	if err != nil {
		return nil, err
	}

	id := conv.UUID(request.UserId)

	dto := &user.PatchParamsDTO{
		FirstName:      request.Body.FirstName,
		LastName:       request.Body.LastName,
		Email:          conv.EmailPtr(request.Body.Email),
		Phone:          request.Body.Phone,
		PostalCode:     request.Body.PostalCode,
		PrefectureName: request.Body.Prefecture,
		City:           request.Body.City,
		Street:         request.Body.Street,
		Building:       request.Body.Building,
	}

	res, err := s.uc.UpdateUserPartially(ctx, &authn, id, dto)
	if err != nil {
		return nil, err
	}

	return gen.PatchUsersDetail200JSONResponse(toUserResponse(res)), nil
}

// DeleteUsersDetail は、指定されたUUIDに該当するユーザーを論理削除します。
func (s *server) DeleteUsersDetail(ctx context.Context, request gen.DeleteUsersDetailRequestObject) (gen.DeleteUsersDetailResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	authn, err := ctxhelper.RequireAuthn(ctx)
	if err != nil {
		return nil, err
	}

	id := conv.UUID(request.UserId)

	if err := s.uc.DeleteUser(ctx, &authn, id); err != nil {
		return nil, err
	}

	return gen.DeleteUsersDetail204Response{}, nil
}

// toUserResponse は、ユースケースのDTOをHTTPレスポンスへ変換します。
func toUserResponse(dto user.UserView) gen.UserResponse {
	return gen.UserResponse{
		Id:         dto.ID.ToPrimitive(),
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
