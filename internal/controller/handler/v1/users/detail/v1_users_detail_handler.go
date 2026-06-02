//go:generate oapi-codegen --include-tags=v1/users/detail --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/users/detail --package=gen --generate=echo-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package detail は、/v1/users/{user_id} エンドポイントに関連するハンドラを提供します。
package detail

import (
	"context"

	"go-boilerplate/internal/controller/conv"
	"go-boilerplate/internal/controller/handler/v1/users/detail/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/user"

	"github.com/labstack/echo/v4"
	"github.com/oapi-codegen/runtime/types"
)

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

// GetUsersDetail は、指定されたUUIDに該当するユーザーの詳細情報を取得します。
func (s *server) GetUsersDetail(ctx context.Context, request gen.GetUsersDetailRequestObject) (gen.GetUsersDetailResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	id := conv.UUID(request.UserId)

	dto, err := s.uc.GetUser(ctx, id)
	if err != nil {
		return nil, err
	}

	return gen.GetUsersDetail200JSONResponse(toUserResponse(dto)), nil
}

// PutUsersDetail は、指定されたUUIDに該当するユーザーの情報を全て更新します（パスワードを含む）。
func (s *server) PutUsersDetail(ctx context.Context, request gen.PutUsersDetailRequestObject) (gen.PutUsersDetailResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	id := conv.UUID(request.UserId)

	dto := &user.UpdateParamsDTO{
		FirstName:      request.Body.FirstName,
		LastName:       request.Body.LastName,
		Email:          string(request.Body.Email),
		Phone:          request.Body.Phone,
		PostalCode:     request.Body.PostalCode,
		PrefectureName: request.Body.Prefecture,
		City:           request.Body.City,
		Street:         request.Body.Street,
		Building:       request.Body.Building,
		RawPassword:    request.Body.Password,
	}

	res, err := s.uc.UpdateUser(ctx, id, dto)
	if err != nil {
		return nil, err
	}

	return gen.PutUsersDetail200JSONResponse(toUserResponse(res)), nil
}

// PatchUsersDetail は、指定されたUUIDに該当するユーザーの情報を部分的に更新します（パスワードは更新しません）。
func (s *server) PatchUsersDetail(ctx context.Context, request gen.PatchUsersDetailRequestObject) (gen.PatchUsersDetailResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	id := conv.UUID(request.UserId)

	dto := &user.PatchParamsDTO{
		FirstName:      request.Body.FirstName,
		LastName:       request.Body.LastName,
		Email:          emailToStringPtr(request.Body.Email),
		Phone:          request.Body.Phone,
		PostalCode:     request.Body.PostalCode,
		PrefectureName: request.Body.Prefecture,
		City:           request.Body.City,
		Street:         request.Body.Street,
		Building:       request.Body.Building,
	}

	res, err := s.uc.UpdateUserPartially(ctx, id, dto)
	if err != nil {
		return nil, err
	}

	return gen.PatchUsersDetail200JSONResponse(toUserResponse(res)), nil
}

// DeleteUsersDetail は、指定されたUUIDに該当するユーザーを論理削除します。
func (s *server) DeleteUsersDetail(ctx context.Context, request gen.DeleteUsersDetailRequestObject) (gen.DeleteUsersDetailResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	id := conv.UUID(request.UserId)

	if err := s.uc.DeleteUser(ctx, id); err != nil {
		return nil, err
	}

	return gen.DeleteUsersDetail204Response{}, nil
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

// emailToStringPtr は、PATCH リクエストの任意 Email を文字列ポインタへ変換します。
func emailToStringPtr(e *types.Email) *string {
	if e == nil {
		return nil
	}
	s := string(*e)
	return &s
}
