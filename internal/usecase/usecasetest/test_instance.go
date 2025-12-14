// Package usecasetest は、ユースケースのテストに関連するユーティリティを提供します。
package usecasetest

import (
	"context"
	"testing"
	"time"

	"boilerplate-go/internal/observability"
	"boilerplate-go/pkg/xerrors"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func ExpectedDBError(t *testing.T) error {
	t.Helper()
	return xerrors.New("database error")
}

// NewTestInstanceForNew は、ユースケースのNew関数用テストインスタンスを生成します。
func NewTestInstanceForNew(t *testing.T) (
	*gomock.Controller, observability.TracerFactory,
) {
	t.Helper()
	ctrl := gomock.NewController(t)
	tf := observability.NewNoopTracerFactory(t)
	return ctrl, tf
}

// NewTestInstanceForImplementedUsecase は、実装済みユースケース用テストインスタンスを生成します。
func NewTestInstanceForImplementedUsecase(t *testing.T) (
	context.Context, *gomock.Controller, *time.Location, observability.LayerTracer,
) {
	t.Helper()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	location, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err)

	tf := observability.NewNoopTracerFactory(t)
	lt := tf.Usecase()

	return ctx, ctrl, location, lt
}
