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

// errExpectedDB は、テストが注入するデータベースエラーです。
var errExpectedDB = xerrors.New("database error")

// ExpectedDBError は、データベースエラーを表すテスト用のエラーを返します。
// 呼び出しごとに同一のエラーを返すため、注入先の検証は errors.Is で行えます。
func ExpectedDBError() error {
	return errExpectedDB
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
