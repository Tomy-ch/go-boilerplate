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

type roleRepository struct {
	db     driver.DatabaseDriver
	tracer observability.LayerTracer
}

// NewRoleRepository は、user.RoleRepository の RDB 実装を生成して返します。
func NewRoleRepository(
	db driver.DatabaseDriver,
	tf observability.TracerFactory,
) user.RoleRepository {
	return &roleRepository{
		db:     db,
		tracer: tf.Infra(),
	}
}

// FindRolesByUserID は、指定ユーザーに割り当てられた全ロールを取得します。
func (r *roleRepository) FindRolesByUserID(ctx context.Context, userID uuid.UUID) (user.Roles, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))
	rows, err := db.GetUserRolesByUserID(ctx, userID)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	roles := make(user.Roles, len(rows))
	for i, row := range rows {
		role, err := rowToRole(row.ID, row.Name, row.Code)
		if err != nil {
			return nil, err
		}
		roles[i] = role
	}

	return roles, nil
}

// rowToRole は、DB 行の値からロールエンティティを再構築します。
// 再構築時の検証失敗はデータ不整合として ErrInternal へ正規化します。
func rowToRole(id uuid.UUID, name string, code int16) (*user.Role, error) {
	entity, err := user.NewRole(id, name, int(code))
	if err != nil {
		return nil, pgerror.NormalizeReconstructError(err)
	}

	return entity, nil
}
