// Package user は、ユーザーに関するドメインのリポジトリを提供します。
package user

import (
	"context"
	"time"

	"boilerplate-go/internal/domain/user"
	"boilerplate-go/internal/infrastructure/rdb/conv"
	"boilerplate-go/internal/infrastructure/rdb/driver"
	"boilerplate-go/internal/infrastructure/rdb/driver/loggingdb"
	"boilerplate-go/internal/infrastructure/rdb/postgres/pgerror"
	"boilerplate-go/internal/infrastructure/rdb/sqlc"
	"boilerplate-go/internal/infrastructure/rdb/sqlc/gen"
	"boilerplate-go/internal/observability"
)

type repository struct {
	db       driver.DatabaseDriver
	provider loggingdb.DBProvider
	tracer   observability.LayerTracer
}

func New(
	db driver.DatabaseDriver,
	provider loggingdb.DBProvider,
	tf observability.TracerFactory,
) user.Repository {
	return &repository{
		db:       db,
		provider: provider,
		tracer:   tf.Infra(),
	}
}

// FindAll は、全ユーザーの情報を取得します。
func (r *repository) FindAll(ctx context.Context, limit, offset int32) (user.Entities, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(r.provider.NewLoggingDB(ctx))
	rows, err := db.ListUsers(ctx, &gen.ListUsersParams{
		OffsetParam: offset,
		LimitParam:  limit,
	})
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	users := make(user.Entities, len(rows))
	for i, row := range rows {
		user, err := user.New(
			row.Users.ID.String(),
			row.Users.FirstName,
			row.Users.LastName,
			row.Users.PasswordHash,
			row.Users.Email,
			row.Users.Phone,
			row.Users.PrefectureID.String(),
			row.Users.City,
			row.Users.Street,
			conv.StringPtrFromNull(row.Users.Building),
			row.Users.PostalCode,
			conv.TimePtrFromNull(row.Users.DeletedAt),
		)
		if err != nil {
			return nil, err
		}
		users[i] = user
	}
	return users, nil
}

// FindByKeyword は、キーワード検索でユーザーの情報を取得します。
func (r *repository) FindByKeyword(ctx context.Context, keywords []string, active *bool, limit, offset int32) (user.Entities, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	tokens := make([]string, len(keywords))
	for i, kw := range keywords {
		escaped := sqlc.EscapeForLike(kw, sqlc.DefaultLikeEscapeChar)
		tokens[i] = sqlc.WrapContainsLikePattern(escaped)
	}

	db := gen.New(r.provider.NewLoggingDB(ctx))
	rows, err := db.ListUsersByKeywords(ctx, &gen.ListUsersByKeywordsParams{
		PatternsParam: tokens,
		DeletedState:  sqlc.BoolPtrToDeletedState(active),
		LimitParam:    int32(limit),
		OffsetParam:   int32(offset),
	})
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	users := make(user.Entities, len(rows))
	for i, row := range rows {
		user, err := user.New(
			row.Users.ID.String(),
			row.Users.FirstName,
			row.Users.LastName,
			row.Users.PasswordHash,
			row.Users.Email,
			row.Users.Phone,
			row.Users.PrefectureID.String(),
			row.Users.City,
			row.Users.Street,
			conv.StringPtrFromNull(row.Users.Building),
			row.Users.PostalCode,
			conv.TimePtrFromNull(row.Users.DeletedAt),
		)
		if err != nil {
			return nil, err
		}
		users[i] = user
	}
	return users, nil
}

// CreateUser は、ユーザーを作成します。
func (r *repository) CreateUser(ctx context.Context, datetime time.Time, user *user.Entity) error {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(r.provider.NewLoggingDB(ctx))
	err := db.CreateUser(ctx, &gen.CreateUserParams{
		ID:           user.ID().ToPrimitive(),
		FirstName:    user.FirstName(),
		LastName:     user.LastName(),
		PasswordHash: user.Password(),
		Email:        user.Email(),
		Phone:        user.Phone(),
		PrefectureID: user.PrefectureID().ToPrimitive(),
		City:         user.City(),
		Street:       user.Street(),
		Building:     conv.NullStringFromPtr(user.Building()),
		PostalCode:   user.PostalCode(),
		CreatedAt:    datetime,
		UpdatedAt:    datetime,
	})
	if err != nil {
		return pgerror.NormalizeError(err)
	}
	return nil
}
