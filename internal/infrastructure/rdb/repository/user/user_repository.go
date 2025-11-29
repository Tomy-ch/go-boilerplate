// Package user は、ユーザーに関するドメインのリポジトリを提供します。
package user

import (
	"context"
	"database/sql"

	"boilerplate-go/internal/domain/user"
	"boilerplate-go/internal/infrastructure/rdb/conv"
	rdbdriver "boilerplate-go/internal/infrastructure/rdb/driver"
	"boilerplate-go/internal/infrastructure/rdb/sqlc"

	"go.uber.org/zap"
)

type repository struct {
	db *sql.DB
	z  *zap.Logger
}

func New(db *sql.DB, z *zap.Logger) user.Repository {
	return &repository{
		db: db,
		z:  z,
	}
}

// GetAllUsers は、全ユーザーの情報を取得します。
func (r *repository) GetAllUsers(ctx context.Context, limit, offset int) (user.Entities, error) {
	db := sqlc.New(rdbdriver.ResolveDriverWithLog(ctx, r.db, r.z))
	rows, err := db.GetUsersDomain(ctx, sqlc.GetUsersDomainParams{
		OffsetParam: conv.NewNullInt64(int64(offset)),
		LimitParam:  conv.NewNullInt64(int64(limit)),
	})
	if err != nil {
		return nil, err
	}

	users := make(user.Entities, len(rows))
	for i, row := range rows {
		user, err := user.New(
			row.ID.String(),
			row.FirstName,
			row.LastName,
			row.PasswordHash,
			row.Email,
			row.Phone,
			row.PrefectureID.String(),
			row.PrefectureName,
			row.City,
			row.Street,
			conv.StringPtrFromNull(row.Building),
			row.PostalCode,
			conv.TimePtrFromNull(row.DeletedAt),
		)
		if err != nil {
			return nil, err
		}
		users[i] = *user
	}
	return users, nil
}
