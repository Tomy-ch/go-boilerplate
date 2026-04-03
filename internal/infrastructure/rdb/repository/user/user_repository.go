// Package user は、ユーザーに関するドメインのリポジトリを提供します。
package user

import (
	"context"

	"boilerplate-go/internal/domain/user"
	"boilerplate-go/internal/infrastructure/rdb/driver/loggingdb"
	"boilerplate-go/internal/infrastructure/rdb/postgres/pgerror"
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

// FindByActive は、アクティブ状態に基づいてユーザーの情報を取得します。
func (r *repository) FindByActive(ctx context.Context, active *bool, limit, offset int32) (user.Users, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(r.db.NewLoggingDB(ctx))

	switch {
	case active == nil:
		return fetchListUsersRows(ctx, db, &gen.ListUsersParams{
			OffsetParam: offset,
			LimitParam:  limit,
		})
	case *active:
		return fetchListUsersRowsByActive(ctx, db, &gen.ListActiveUsersParams{
			OffsetParam: offset,
			LimitParam:  limit,
		})
	case !*active:
		return fetchListUsersRowsByDeleted(ctx, db, &gen.ListDeletedUsersParams{
			OffsetParam: offset,
			LimitParam:  limit,
		})
	default:
		panic("unreachable: invalid active")
	}
}

// FindByDeleted は、削除されたユーザーの情報を取得します。
func fetchListUsersRows(
	ctx context.Context, db *gen.Queries, params *gen.ListUsersParams,
) (user.Users, error) {
	rows, err := db.ListUsers(ctx, params)
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

// fetchListUsersRowsByActive は、アクティブ状態に基づいてユーザーの情報を取得します。
func fetchListUsersRowsByActive(
	ctx context.Context, db *gen.Queries, params *gen.ListActiveUsersParams,
) (user.Users, error) {
	rows, err := db.ListActiveUsers(ctx, params)
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

// fetchListUsersRowsByDeleted は、削除されたユーザーの情報を取得します。
func fetchListUsersRowsByDeleted(
	ctx context.Context, db *gen.Queries, params *gen.ListDeletedUsersParams,
) (user.Users, error) {
	rows, err := db.ListDeletedUsers(ctx, params)
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
	switch {
	case active == nil:
		return db.CountUsers(ctx)
	case *active:
		return db.CountActiveUsers(ctx)
	case !*active:
		return db.CountDeletedUsers(ctx)
	default:
		panic("unreachable: invalid active")
	}
}
