// Package prefecture は、都道府県関連のドメインを提供します。
package prefecture

import (
	"context"

	"boilerplate-go/internal/domain/prefecture"
	"boilerplate-go/internal/infrastructure/rdb/driver/loggingdb"
	"boilerplate-go/internal/infrastructure/rdb/postgres/pgerror"
	"boilerplate-go/internal/infrastructure/rdb/sqlc/gen"
	"boilerplate-go/internal/observability"
	"boilerplate-go/pkg/uuid"
)

type repository struct {
	provider loggingdb.DBProvider
	tracer   observability.LayerTracer
}

func New(
	provider loggingdb.DBProvider,
	tf observability.TracerFactory,
) prefecture.Repository {
	return &repository{
		provider: provider,
		tracer:   tf.Infra(),
	}
}

// FindByID は、IDから都道府県エンティティを取得します。
func (r *repository) FindByID(ctx context.Context, id uuid.UUID) (*prefecture.Entity, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(r.provider.NewLoggingDB(ctx))
	row, err := db.GetPrefectureDomainByID(ctx, id.ToPrimitive())
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	return prefecture.New(
		row.ID.String(),
		row.Name,
		int(row.Code),
	)
}

// FindByIDs は、複数IDから都道府県エンティティ一覧を取得します。
func (r *repository) FindByIDs(ctx context.Context, ids []uuid.UUID) (prefecture.Entities, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(r.provider.NewLoggingDB(ctx))
	rows, err := db.GetPrefectureDomainByIDs(ctx, uuid.ToPrimitiveUniqueList(ids))
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	prefectures := make(prefecture.Entities, len(rows))
	for i, row := range rows {
		prefectureEntity, err := prefecture.New(
			row.ID.String(),
			row.Name,
			int(row.Code),
		)
		if err != nil {
			return nil, pgerror.NormalizeError(err)
		}
		prefectures[i] = prefectureEntity
	}

	return prefectures, nil
}

// FindByName は、都道府県名から都道府県エンティティを取得します。
func (r *repository) FindByName(ctx context.Context, name string) (*prefecture.Entity, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(r.provider.NewLoggingDB(ctx))
	row, err := db.GetPrefectureDomainByName(ctx, name)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	return prefecture.New(
		row.ID.String(),
		row.Name,
		int(row.Code),
	)
}
