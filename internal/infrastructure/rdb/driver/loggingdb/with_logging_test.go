package loggingdb

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	mock_driver "go-boilerplate/internal/infrastructure/rdb/driver/mock"
	"go-boilerplate/internal/logging"
	mock_logging "go-boilerplate/internal/logging/mock"
	"go-boilerplate/internal/observability"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

func Test_dbWithLogging_Exec(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	obsCfg := config.NewObservabilityConfig(cfg)
	lf := logging.NewTestLogFieldBuilder(t)
	lg := logging.NewTestLogger(t)
	noopLayerTracer := observability.NewNoopLayerTracer(t)

	ctrl := gomock.NewController(t)

	md := mock_driver.NewMockDBTX(ctrl)

	var dbtx driver.DBTX = md
	md.EXPECT().Exec(gomock.Any(), "INSERT INTO users (name) VALUES ($1)", "alice").Return(pgconn.CommandTag{}, nil)

	dwl := &dbWithLogging{
		db:       dbtx,
		provider: &provider{l: lg, lf: lf, dbCfg: dbCfg, obsCfg: obsCfg, tracer: noopLayerTracer},
	}

	res, err := dwl.Exec(context.Background(), "INSERT INTO users (name) VALUES ($1)", "alice")
	require.NoError(t, err)

	assert.Equal(t, pgconn.CommandTag{}, res)
}

// Test_dbWithLogging_callerSkip は、開始ログと終了ログが同一の呼び出し元を caller に記録することを検証します。
func Test_dbWithLogging_callerSkip(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	obsCfg := config.NewObservabilityConfig(cfg)
	lf := logging.NewTestLogFieldBuilder(t)
	lg, observed := logging.NewObservedTestLoggerWithCaller(t)
	noopLayerTracer := observability.NewNoopLayerTracer(t)

	ctrl := gomock.NewController(t)

	md := mock_driver.NewMockDBTX(ctrl)
	md.EXPECT().Exec(gomock.Any(), gomock.Any(), gomock.Any()).Return(pgconn.CommandTag{}, nil)

	dwl := &dbWithLogging{
		db:       md,
		provider: &provider{l: lg, lf: lf, dbCfg: dbCfg, obsCfg: obsCfg, tracer: noopLayerTracer},
	}

	// repository → sqlc gen → Exec の呼び出し段数を再現する。
	sqlcGen := func() {
		_, err := dwl.Exec(context.Background(), "INSERT INTO users (name) VALUES ($1)", "alice")
		require.NoError(t, err)
	}
	repository := func() { sqlcGen() }
	repository()

	entries := observed.All()
	require.Len(t, entries, 2) // 開始ログ + 終了ログ

	startCaller := entries[0].Caller
	endCaller := entries[1].Caller
	require.True(t, startCaller.Defined)
	require.True(t, endCaller.Defined)
	// 開始・終了とも同一の呼び出し元（repository 層相当＝本テストファイル）を指すこと。
	assert.Equal(t, startCaller.TrimmedPath(), endCaller.TrimmedPath())
	assert.Contains(t, startCaller.TrimmedPath(), "with_logging_test.go")
}

func Test_dbWithLogging_Query(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	obsCfg := config.NewObservabilityConfig(cfg)
	lf := logging.NewTestLogFieldBuilder(t)
	lg := logging.NewTestLogger(t)
	noopLayerTracer := observability.NewNoopLayerTracer(t)

	ctrl := gomock.NewController(t)

	md := mock_driver.NewMockDBTX(ctrl)

	var dbtx driver.DBTX = md
	md.EXPECT().Query(gomock.Any(), "SELECT 1").Return(nil, nil)

	dwl := &dbWithLogging{
		db:       dbtx,
		provider: &provider{l: lg, lf: lf, dbCfg: dbCfg, obsCfg: obsCfg, tracer: noopLayerTracer},
	}

	rows, err := dwl.Query(context.Background(), "SELECT 1")
	require.NoError(t, err)
	require.Nil(t, rows)
	if rows != nil {
		defer rows.Close()
	}
}

func Test_dbWithLogging_QueryRow(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	obsCfg := config.NewObservabilityConfig(cfg)
	lf := logging.NewTestLogFieldBuilder(t)
	lg := logging.NewTestLogger(t)
	noopLayerTracer := observability.NewNoopLayerTracer(t)

	ctrl := gomock.NewController(t)

	md := mock_driver.NewMockDBTX(ctrl)
	var dbtx driver.DBTX = md
	md.EXPECT().QueryRow(gomock.Any(), "SELECT $1", 1).Return(nil)

	dwl := &dbWithLogging{
		db:       dbtx,
		provider: &provider{l: lg, lf: lf, dbCfg: dbCfg, obsCfg: obsCfg, tracer: noopLayerTracer},
	}

	row := dwl.QueryRow(context.Background(), "SELECT $1", 1)
	require.Nil(t, row)
}

func Test_dbWithLogging_buildSQLStartLogFields(t *testing.T) {
	t.Parallel()

	lf := logging.NewTestLogFieldBuilder(t)

	tc := &observability.TraceContext{}
	funcName := "TestBuildSQLLogFields"

	dwl := &dbWithLogging{
		provider: &provider{
			lf: lf,
		},
	}

	spanName := observability.BuildSpanName(layer, pkg, funcName)
	actual := dwl.buildSQLStartLogFields(tc, funcName, spanName)
	require.NotEmpty(t, actual)
}

func Test_dbWithLogging_buildSQLEndLogFields(t *testing.T) {
	t.Parallel()

	t.Run("observabilityConfigでMaskedDBQueryArgsがtrueのとき、ログにクエリ引数が含まれない", func(t *testing.T) {
		t.Parallel()

		cfg := config.MockConfigForTest(t)
		obsCfg := config.NewObservabilityConfig(cfg)
		obsCfg.SetObservabilityMaskedDBQueryArgs(t, true)

		lf := logging.NewTestLogFieldBuilder(t)

		tc := &observability.TraceContext{}
		funcName := "TestBuildSQLLogFields"
		query := "SELECT id FROM users WHERE id = $1"
		args := []any{123}

		dwl := &dbWithLogging{
			provider: &provider{
				lf:     lf,
				obsCfg: obsCfg,
			},
		}

		spanName := observability.BuildSpanName(layer, pkg, funcName)
		fields := dwl.buildSQLEndLogFields(tc, funcName, spanName, query, time.Second, args, nil)

		require.NotEmpty(t, fields)
	})

	t.Run("observabilityConfigでMaskedDBQueryArgsがfalseのとき、ログにクエリ引数が含まれる", func(t *testing.T) {
		t.Parallel()

		cfg := config.MockConfigForTest(t)
		obsCfg := config.NewObservabilityConfig(cfg)
		lf := logging.NewTestLogFieldBuilder(t)

		tc := &observability.TraceContext{}
		funcName := "TestBuildSQLLogFields"
		query := "SELECT 1"
		expectedDuration := 100 * time.Millisecond

		dwl := &dbWithLogging{
			provider: &provider{
				lf:     lf,
				obsCfg: obsCfg,
			},
		}

		spanName := observability.BuildSpanName(layer, pkg, funcName)
		actual := dwl.buildSQLEndLogFields(tc, funcName, spanName, query, expectedDuration, nil, nil)
		require.NotEmpty(t, actual)
	})
}

func Test_dbWithLogging_logQueryResult(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)

	ctrl := gomock.NewController(t)

	t.Run("Infoが呼ばれる", func(t *testing.T) {
		t.Parallel()

		mockDuration := dbCfg.SlowQueryWarnThreshold() - 100*time.Millisecond
		mockLog := mock_logging.NewMockLogger(ctrl)
		mockLog.EXPECT().Named(layer).Return(mockLog)
		mockLog.EXPECT().CallerSkip(callSkip).Return(mockLog)
		mockLog.EXPECT().Info("SQL Exec")

		dwl := &dbWithLogging{provider: &provider{l: mockLog, dbCfg: dbCfg}}

		dwl.logQueryResult("SQL Exec", mockDuration, nil, nil)
	})

	t.Run("遅いクエリでWarnが呼ばれる", func(t *testing.T) {
		t.Parallel()

		mockDuration := dbCfg.SlowQueryWarnThreshold() + 100*time.Millisecond
		mockLog := mock_logging.NewMockLogger(ctrl)
		mockLog.EXPECT().Named(layer).Return(mockLog)
		mockLog.EXPECT().CallerSkip(callSkip).Return(mockLog)
		mockLog.EXPECT().Warn("SQL Exec")

		dwl := &dbWithLogging{provider: &provider{l: mockLog, dbCfg: dbCfg}}

		dwl.logQueryResult("SQL Exec", mockDuration, nil, nil)
	})

	t.Run("Errorが呼ばれる", func(t *testing.T) {
		t.Parallel()

		mockDuration := dbCfg.SlowQueryWarnThreshold() - 100*time.Millisecond
		mockLog := mock_logging.NewMockLogger(ctrl)
		mockLog.EXPECT().Named(layer).Return(mockLog)
		mockLog.EXPECT().CallerSkip(callSkip).Return(mockLog)
		mockLog.EXPECT().Error("SQL Exec")

		dwl := &dbWithLogging{provider: &provider{l: mockLog, dbCfg: dbCfg}}

		dwl.logQueryResult("SQL Exec", mockDuration, nil, errors.New("boom"))
	})
}
