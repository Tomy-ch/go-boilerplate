// Package user は、ユーザーに関するドメインのリポジトリを提供します。
package user

import (
	"context"

	"boilerplate-go/internal/domain/user"
	"boilerplate-go/internal/infrastructure/rdb/driver/loggingdb"
	"boilerplate-go/internal/infrastructure/rdb/postgres/pgerror"
	"boilerplate-go/internal/infrastructure/rdb/sqlc"
	"boilerplate-go/internal/infrastructure/rdb/sqlc/gen"
	"boilerplate-go/internal/observability"
)

type repository struct {
	db     loggingdb.DBProvider
	tracer observability.LayerTracer
}

func New(
	db loggingdb.DBProvider,
	tf observability.TracerFactory,
) user.Repository {
	return &repository{
		db:     db,
		tracer: tf.Infra(),
	}
}

// FindAll は、全ユーザーの情報を取得します。
func (r *repository) FindAll(ctx context.Context, limit, offset int32) (user.Users, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(r.db.NewLoggingDB(ctx))
	rows, err := db.ListUsers(ctx, &gen.ListUsersParams{
		OffsetParam: offset,
		LimitParam:  limit,
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

// Create は、ユーザーを作成します。
func (r *repository) Create(ctx context.Context, user *user.User) error {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(r.db.NewLoggingDB(ctx))
	err := db.CreateUser(ctx, &gen.CreateUserParams{
		ID:           user.ID(),
		FirstName:    user.FirstName(),
		LastName:     user.LastName(),
		PasswordHash: user.PasswordHash(),
		Email:        user.Email(),
		Phone:        user.Phone(),
		PrefectureID: user.PrefectureID(),
		City:         user.City(),
		Street:       user.Street(),
		Building:     user.Building(),
		PostalCode:   user.PostalCode(),
		CreatedAt:    user.CreatedAt(),
		UpdatedAt:    user.UpdatedAt(),
	})
	if err != nil {
		return pgerror.NormalizeError(err)
	}
	return nil
}

// CountByActive は、アクティブ状態に基づいてユーザーの総件数を返します。
func (r *repository) CountByActive(ctx context.Context, active *bool) (int64, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(r.db.NewLoggingDB(ctx))
	count, err := db.CountUsersByDeletedState(ctx, sqlc.BoolPtrToDeletedState(active))
	if err != nil {
		return 0, pgerror.NormalizeError(err)
	}
	return count, nil
}
