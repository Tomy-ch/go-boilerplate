// Package testkit は、ユースケースのテストに関連するユーティリティを提供します。
package testkit

import (
	"context"
	"testing"

	"go-boilerplate/internal/usecase/boundary/tx"
	mock_tx "go-boilerplate/internal/usecase/boundary/tx/mock"
	"go-boilerplate/pkg/xerrors"

	"go.uber.org/mock/gomock"
)

// ExpectedDBError は、データベースエラーを表すテスト用のエラーを生成します。
func ExpectedDBError() error {
	return xerrors.New("database error")
}

// NewMockTransactionManager は、テスト用のトランザクションマネージャーを生成します。
func NewMockTransactionManager(t *testing.T) tx.Manager {
	t.Helper()
	ctrl := gomock.NewController(t)
	mockTx := mock_tx.NewMockManager(ctrl)
	mockTx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, fn func(ctx context.Context) error) error {
			return fn(ctx)
		},
	).AnyTimes()
	return mockTx
}
