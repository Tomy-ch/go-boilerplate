// Package user は、ユーザーに関するドメインのリポジトリを提供します。
package user

import (
	"context"

	"go-boilerplate/internal/domain/user"
	"go-boilerplate/internal/infrastructure/rdb/driver/loggingdb"
	"go-boilerplate/internal/infrastructure/rdb/pgerror"
	"go-boilerplate/internal/infrastructure/rdb/sqlc/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/pkg/uuid"
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

// rowToUser は、sqlc が返す Users 行をドメインエンティティへ変換します。
func rowToUser(u gen.Users) (*user.User, error) {
	return user.New(
		u.ID,
		u.FirstName,
		u.LastName,
		u.PasswordHash,
		u.Email,
		u.Phone,
		u.PrefectureID,
		u.City,
		u.Street,
		u.Building,
		u.PostalCode,
		u.CreatedAt,
		u.UpdatedAt,
		u.DeletedAt,
	)
}

// rowsToUsers は、行スライスをドメインエンティティ列へ変換します。
func rowsToUsers[T any](rows []T, extract func(T) gen.Users) (user.Users, error) {
	users := make(user.Users, len(rows))
	for i, row := range rows {
		u, err := rowToUser(extract(row))
		if err != nil {
			return nil, err
		}
		users[i] = u
	}
	return users, nil
}

// fetchListUsersRows は、ユーザーの情報を取得します。
func fetchListUsersRows(
	ctx context.Context, db *gen.Queries, params *gen.ListUsersParams,
) (user.Users, error) {
	rows, err := db.ListUsers(ctx, params)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}
	return rowsToUsers(rows, func(r *gen.ListUsersRow) gen.Users { return r.Users })
}

// fetchListUsersRowsByActive は、アクティブ状態に基づいてユーザーの情報を取得します。
func fetchListUsersRowsByActive(
	ctx context.Context, db *gen.Queries, params *gen.ListActiveUsersParams,
) (user.Users, error) {
	rows, err := db.ListActiveUsers(ctx, params)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}
	return rowsToUsers(rows, func(r *gen.ListActiveUsersRow) gen.Users { return r.Users })
}

// fetchListUsersRowsByDeleted は、削除されたユーザーの情報を取得します。
func fetchListUsersRowsByDeleted(
	ctx context.Context, db *gen.Queries, params *gen.ListDeletedUsersParams,
) (user.Users, error) {
	rows, err := db.ListDeletedUsers(ctx, params)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}
	return rowsToUsers(rows, func(r *gen.ListDeletedUsersRow) gen.Users { return r.Users })
}

// Create は、ユーザーを作成します。
func (r *repository) Create(ctx context.Context, u *user.User) error {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(r.db.NewLoggingDB(ctx))
	err := db.CreateUser(ctx, &gen.CreateUserParams{
		ID:           u.ID(),
		FirstName:    u.FirstName(),
		LastName:     u.LastName(),
		PasswordHash: u.PasswordHash(),
		Email:        u.Email(),
		Phone:        u.Phone(),
		PrefectureID: u.PrefectureID(),
		City:         u.City(),
		Street:       u.Street(),
		Building:     u.Building(),
		PostalCode:   u.PostalCode(),
		CreatedAt:    u.CreatedAt(),
		UpdatedAt:    u.UpdatedAt(),
	})
	if err != nil {
		return pgerror.NormalizeError(err)
	}
	return nil
}

// FindByID は、IDから単一ユーザーを取得します。存在しない場合は NotFound に正規化したエラーを返します。
func (r *repository) FindByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(r.db.NewLoggingDB(ctx))
	row, err := db.GetUserByID(ctx, id)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	return rowToUser(row.Users)
}

// Update は、ユーザーの mutable フィールドと updatedAt / deletedAt を更新します。
func (r *repository) Update(ctx context.Context, u *user.User) error {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(r.db.NewLoggingDB(ctx))
	rows, err := db.UpdateUser(ctx, &gen.UpdateUserParams{
		FirstName:    u.FirstName(),
		LastName:     u.LastName(),
		PasswordHash: u.PasswordHash(),
		Email:        u.Email(),
		Phone:        u.Phone(),
		PrefectureID: u.PrefectureID(),
		City:         u.City(),
		Street:       u.Street(),
		Building:     u.Building(),
		PostalCode:   u.PostalCode(),
		UpdatedAt:    u.UpdatedAt(),
		DeletedAt:    u.DeletedAt(),
		ID:           u.ID(),
	})
	// エラー正規化と「影響行数 0 → NotFound」判定は pgerror に集約
	return pgerror.NormalizeExecResult(rows, err)
}

// CountByActive は、アクティブ状態に基づいてユーザーの総件数を返します。
func (r *repository) CountByActive(ctx context.Context, active *bool) (int64, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(r.db.NewLoggingDB(ctx))

	var (
		count int64
		err   error
	)

	switch {
	case active == nil:
		count, err = db.CountUsers(ctx)
	case *active:
		count, err = db.CountActiveUsers(ctx)
	case !*active:
		count, err = db.CountDeletedUsers(ctx)
	default:
		panic("unreachable: invalid active")
	}

	if err != nil {
		return 0, pgerror.NormalizeError(err)
	}

	return count, nil
}
