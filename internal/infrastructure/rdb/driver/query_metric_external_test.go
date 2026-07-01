package driver_test

import (
	"context"
	"testing"

	"github.com/exaring/otelpgx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	mock_driver "go-boilerplate/internal/infrastructure/rdb/driver/mock"
	"go-boilerplate/internal/logging"
	mock_logging "go-boilerplate/internal/logging/mock"
)

// query metrics の低カーディナリティ enum 値。driver パッケージの未公開定数と同値を外部テストで明示する。
const (
	testOperationSelect = "select"
	testStatusSuccess   = "success"
	testStatusError     = "error"
)

// TestTracedDB_QueryMetricsInstrumentation は、実 DB を用いた黒箱テストで、
// recorder（生成モック MockQueryRecorder）にクエリ属性が記録されること、
// および成功/失敗で status / error_class が適切に分類されることを検証する。
// ラベルに SQL 本文やテーブル名が混入しないことも併せて確認する。
func TestTracedDB_QueryMetricsInstrumentation(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	osCfg := config.NewOperatingSystemConfig(cfg)
	dbConnCfg := config.NewDBConnectionConfig(cfg)
	obsCfg := config.NewObservabilityConfig(cfg)

	ctrl := gomock.NewController(t)
	mockLogger := mock_logging.NewMockLogger(ctrl)
	mockLogger.EXPECT().Named(gomock.Any()).Return(mockLogger).AnyTimes()
	mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Error(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Warn(gomock.Any(), gomock.Any()).AnyTimes()
	lf := logging.NewTestLogFieldBuilder(t)

	recorder := mock_driver.NewMockQueryRecorder(ctrl)
	var observed []driver.QueryAttrs
	// db.Exec は逐次実行されるため Observe も逐次呼び出しとなる（追加の排他は不要）。
	recorder.EXPECT().
		Observe(gomock.Any(), gomock.Any()).
		MinTimes(2).
		Do(func(_ context.Context, attrs driver.QueryAttrs) {
			observed = append(observed, attrs)
		})

	tracer := driver.NewQueryTracer(dbCfg, obsCfg, otelpgx.NewTracer(), recorder, mockLogger, lf)

	db, err := driver.NewTracedDB(dbCfg, osCfg, dbConnCfg, tracer)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	ctx := driver.WithQueryName(context.Background(), "test.select_one")

	// 正常クエリ。
	_, err = db.Exec(ctx, "SELECT 1")
	require.NoError(t, err)

	// 失敗クエリ。
	_, err = db.Exec(ctx, "SELECT 1 FROM no_such_table_for_test")
	require.Error(t, err)

	require.GreaterOrEqual(t, len(observed), 2)

	var sawSuccess, sawError bool
	for _, attrs := range observed {
		// query_name / operation は明示名・固定 enum のみで、SQL 本文やテーブル名は混入しない。
		assert.Equal(t, "test.select_one", attrs.QueryName)
		assert.Equal(t, testOperationSelect, attrs.Operation)
		assert.NotContains(t, attrs.QueryName, "SELECT")
		assert.NotContains(t, attrs.Operation, "no_such_table_for_test")

		switch attrs.Status {
		case testStatusSuccess:
			sawSuccess = true
			assert.Empty(t, attrs.ErrorClass)
		case testStatusError:
			sawError = true
			assert.NotEmpty(t, attrs.ErrorClass)
		}
	}
	assert.True(t, sawSuccess, "成功クエリの属性が記録されていること")
	assert.True(t, sawError, "失敗クエリの属性が記録されていること")
}
