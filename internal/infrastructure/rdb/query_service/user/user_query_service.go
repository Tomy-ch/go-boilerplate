// Package user は、ユーザーに関するクエリーサービスを提供します。
package user

import (
	"context"

	"boilerplate-go/internal/domain/user"
	"boilerplate-go/internal/infrastructure/rdb/driver/loggingdb"
	"boilerplate-go/internal/infrastructure/rdb/postgres/pgerror"
	"boilerplate-go/internal/infrastructure/rdb/sqlc"
	"boilerplate-go/internal/infrastructure/rdb/sqlc/gen"
	"boilerplate-go/internal/observability"
	"boilerplate-go/internal/usecase/user/query"
)

type service struct {
	db     loggingdb.DBProvider
	tracer observability.LayerTracer
}

func New(
	db loggingdb.DBProvider,
	tf observability.TracerFactory,
) query.UserQueryService {
	return &service{
		db:     db,
		tracer: tf.Infra(),
	}
}

// FindByKeyword は、キーワード検索でユーザーの情報を取得します。
func (s *service) FindByKeyword(ctx context.Context, keywords []string, active *bool, limit, offset int32) (user.Users, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	tokens := make([]string, len(keywords))
	for i, kw := range keywords {
		escaped := sqlc.EscapeForLike(kw, sqlc.DefaultLikeEscapeChar)
		tokens[i] = sqlc.WrapContainsLikePattern(escaped)
	}

	db := gen.New(s.db.NewLoggingDB(ctx))
	rows, err := db.ListUsersByKeywords(ctx, &gen.ListUsersByKeywordsParams{
		PatternsParam: tokens,
		DeletedState:  sqlc.BoolPtrToDeletedState(active),
		LimitParam:    int32(limit),
		OffsetParam:   int32(offset),
	})
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	users := make(user.Users, len(rows))
	for i, row := range rows {
		user, err := user.New(
			row.Users.ID,
			row.Users.FirstName,
			row.Users.LastName,
			row.Users.PasswordHash,
			row.Users.Email,
			row.Users.Phone,
			row.Users.PrefectureID,
			row.Users.City,
			row.Users.Street,
			row.Users.Building,
			row.Users.PostalCode,
			row.Users.CreatedAt,
			row.Users.UpdatedAt,
			row.Users.DeletedAt,
		)
		if err != nil {
			return nil, err
		}
		users[i] = user
	}
	return users, nil
}
