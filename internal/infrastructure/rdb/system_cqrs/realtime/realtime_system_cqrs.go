// Package realtime は、Realtime Delivery の採番境界（boundary realtime.SequenceAllocator）の
// RDB 実装を提供します。
package realtime

import (
	"context"

	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/pgerror"
	"go-boilerplate/internal/infrastructure/rdb/sqlc/gen"
	"go-boilerplate/internal/observability"
	realtimebndry "go-boilerplate/internal/usecase/boundary/realtime"
	"go-boilerplate/pkg/xerrors"

	"github.com/jackc/pgx/v5"
)

type allocator struct {
	db     driver.DatabaseDriver
	tracer observability.LayerTracer
}

// NewSequenceAllocator は、realtime.SequenceAllocator の RDB 実装を生成して返します。
func NewSequenceAllocator(
	provider driver.DatabaseDriver,
	tf observability.TracerFactory,
) realtimebndry.SequenceAllocator {
	return &allocator{
		db:     provider,
		tracer: tf.Infra(),
	}
}

// Allocate は、ストリームの次の位置を採番して返します。
// 採番行のロックは呼び出し側 tx の commit まで保持されるため、同一ストリームの採番は直列化されます。
func (a *allocator) Allocate(ctx context.Context, streamID realtimebndry.StreamID) (realtimebndry.Sequence, error) {
	ctx, endSpan := a.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, a.db))
	seq, err := db.AllocateStreamSequence(ctx, streamID.String())
	if err != nil {
		return 0, pgerror.NormalizeError(err)
	}
	return realtimebndry.Sequence(seq), nil
}

// Current は、ストリームの現在位置を返します。まだ採番されていなければ ok=false を返します。
func (a *allocator) Current(ctx context.Context, streamID realtimebndry.StreamID) (realtimebndry.Sequence, bool, error) {
	ctx, endSpan := a.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, a.db))
	seq, err := db.CurrentStreamSequence(ctx, streamID.String())
	if err != nil {
		if xerrors.Is(err, pgx.ErrNoRows) {
			return 0, false, nil
		}
		return 0, false, pgerror.NormalizeError(err)
	}
	return realtimebndry.Sequence(seq), true, nil
}
