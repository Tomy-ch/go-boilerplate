// Package user は、ユーザーに関するドメインのリポジトリを提供します。
package user

import (
	"context"

	"boilerplate-go/internal/domain/user"
	"boilerplate-go/internal/infrastructure/rdb/conv"
	"boilerplate-go/internal/infrastructure/rdb/driver"
	"boilerplate-go/internal/infrastructure/rdb/sqlc"
	"boilerplate-go/internal/observability"
)

type repository struct {
	db       driver.DatabaseDriver
	provider driver.LoggingDBProvider
	tracer   observability.LayerTracer
}

func New(db driver.DatabaseDriver, provider driver.LoggingDBProvider, tf observability.TracerFactory) user.Repository {
	return &repository{
		db:       db,
		provider: provider,
		tracer:   tf.Infra(),
	}
}

// GetAllUsers は、全ユーザーの情報を取得します。
func (r *repository) GetAllUsers(ctx context.Context, limit, offset int) (user.Entities, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := sqlc.New(r.provider.NewLoggingDB(ctx))
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
