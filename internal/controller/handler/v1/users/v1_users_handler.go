//go:generate oapi-codegen --include-tags=v1/users --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/users --package=gen --generate=echo-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package v1users は、/v1/users エンドポイントに関連するハンドラを提供します。
package v1users

import (
	"database/sql"
	"net/http"

	"boilerplate-go/internal/controller/handler/v1/users/gen"
	userrepo "boilerplate-go/internal/infrastructure/rdb/repository/user"

	"github.com/labstack/echo/v4"
)

type server struct {
	db *sql.DB
}

func BindHandler(e *echo.Echo, db *sql.DB) {
	gen.RegisterHandlers(e, &server{
		db: db,
	})
}

// GetUsers は、ユーザー一覧を取得します。
func (s *server) GetUsers(ctx echo.Context, _ gen.GetUsersParams) error {
	// WARN: 本来はここで認可を行うべきですが、今回は省略します。
	results, err := userrepo.New(s.db).GetAllUsers(ctx.Request().Context())
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, results)
}

// PostUsers implements gen.ServerInterface.
func (s *server) PostUsers(_ echo.Context) error {
	panic("unimplemented")
}
