package user

import (
	"context"

	"go-boilerplate/internal/domain/user"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/pgerror"
	"go-boilerplate/internal/infrastructure/rdb/sqlc/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/pkg/uuid"
)

type lockRepository struct {
	db     driver.DatabaseDriver
	tracer observability.LayerTracer
}

// NewLockRepository は、user.LockRepository の RDB 実装を生成して返します。
func NewLockRepository(
	db driver.DatabaseDriver,
	tf observability.TracerFactory,
) user.LockRepository {
	return &lockRepository{
		db:     db,
		tracer: tf.Infra(),
	}
}

// LockByID は、未削除の単一ユーザーを悲観ロック（排他）して取得します。
// 論理削除済み・不存在はいずれも 0 行となり NotFound に正規化したエラーを返します。
func (r *lockRepository) LockByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))
	row, err := db.LockUserByID(ctx, id)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	return rowToUser(row.Users)
}

// LockShareByID は、単一ユーザーを共有ロック（FOR SHARE）して取得します。退会済みで除外しないため、
// 在籍していないユーザーもそのまま返します。不存在は 0 行となり NotFound に正規化したエラーを返します。
func (r *lockRepository) LockShareByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))
	row, err := db.LockUserShareByID(ctx, id)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	return rowToUser(row.Users)
}
