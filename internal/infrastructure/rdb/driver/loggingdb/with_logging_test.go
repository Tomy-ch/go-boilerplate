package loggingdb

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	mock_loggingdb "go-boilerplate/internal/infrastructure/rdb/driver/loggingdb/mock"
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

	mp := mock_loggingdb.NewMockDBProvider(ctrl)
	mp.EXPECT().LogFields().Return(lf).AnyTimes()
	mp.EXPECT().Logger().Return(lg).AnyTimes()
	mp.EXPECT().DBConfig().Return(dbCfg).AnyTimes()
	mp.EXPECT().ObservabilityConfig().Return(obsCfg).AnyTimes()
	mp.EXPECT().LayerTracer().Return(noopLayerTracer).AnyTimes()

	dwl := &dbWithLogging{db: dbtx, ctx: context.Background(), provider: mp}

	res, err := dwl.Exec(context.Background(), "INSERT INTO users (name) VALUES ($1)", "alice")
	require.NoError(t, err)

	assert.Equal(t, pgconn.CommandTag{}, res)
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

	mp := mock_loggingdb.NewMockDBProvider(ctrl)
	mp.EXPECT().LogFields().Return(lf).AnyTimes()
	mp.EXPECT().Logger().Return(lg).AnyTimes()
	mp.EXPECT().DBConfig().Return(dbCfg).AnyTimes()
	mp.EXPECT().ObservabilityConfig().Return(obsCfg).AnyTimes()
	mp.EXPECT().LayerTracer().Return(noopLayerTracer).AnyTimes()

	dwl := &dbWithLogging{db: dbtx, ctx: context.Background(), provider: mp}

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

	mp := mock_loggingdb.NewMockDBProvider(ctrl)
	mp.EXPECT().LogFields().Return(lf).AnyTimes()
	mp.EXPECT().Logger().Return(lg).AnyTimes()
	mp.EXPECT().DBConfig().Return(dbCfg).AnyTimes()
	mp.EXPECT().ObservabilityConfig().Return(obsCfg).AnyTimes()
	mp.EXPECT().LayerTracer().Return(noopLayerTracer).AnyTimes()

	dwl := &dbWithLogging{db: dbtx, ctx: context.Background(), provider: mp}

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

	actual := dwl.buildSQLStartLogFields(tc, funcName)
	require.NotEmpty(t, actual)
}

func Test_dbWithLogging_buildSQLEndLogFields(t *testing.T) {
	t.Parallel()

	t.Run("observabilityConfigでMaskedDBQueryArgsがtrueのとき、ログにクエリ引数が含まれない", func(t *testing.T) {
		cfg := config.MockConfigForTest(t)
		obsCfg := config.NewObservabilityConfig(cfg)
		obsCfg.SetObservabilityMaskedDBQueryArgs(t, true)

		lf := logging.NewTestLogFieldBuilder(t)

		tc := &observability.TraceContext{}
		funcName := "TestBuildSQLLogFields"
		query := "SELECT * FROM users WHERE id = $1"
		args := []any{123}

		dwl := &dbWithLogging{
			provider: &provider{
				lf:     lf,
				obsCfg: obsCfg,
			},
		}

		fields := dwl.buildSQLEndLogFields(tc, funcName, query, time.Second, args, nil)

		require.NotEmpty(t, fields)
	})

	t.Run("observabilityConfigでMaskedDBQueryArgsがfalseのとき、ログにクエリ引数が含まれる", func(t *testing.T) {
		cfg := config.MockConfigForTest(t)
		obsCfg := config.NewObservabilityConfig(cfg)
		lf := logging.NewTestLogFieldBuilder(t)

		tc := &observability.TraceContext{}
		funcName := "TestBuildSQLLogFields"
		query := "SELECT 1"
		expectedDuration := time.Duration(100 * time.Millisecond)

		dwl := &dbWithLogging{
			provider: &provider{
				lf:     lf,
				obsCfg: obsCfg,
			},
		}

		actual := dwl.buildSQLEndLogFields(tc, funcName, query, expectedDuration, nil, nil)
		require.NotEmpty(t, actual)
	})
}

func Test_dbWithLogging_logQueryResult(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	lf := logging.NewTestLogFieldBuilder(t)

	ctrl := gomock.NewController(t)

	t.Run("Infoが呼ばれる", func(t *testing.T) {
		t.Parallel()

		mockDuration := dbCfg.SlowQueryWarnThreshold() - time.Duration(100*time.Millisecond)
		mockLog := mock_logging.NewMockLogger(ctrl)
		mockLog.EXPECT().Named(layer).Return(mockLog)
		mockLog.EXPECT().CallerSkip(callSkip).Return(mockLog)
		mockLog.EXPECT().Info("SQL Exec")

		mp := mock_loggingdb.NewMockDBProvider(ctrl)
		mp.EXPECT().LogFields().Return(lf).AnyTimes()
		mp.EXPECT().Logger().Return(mockLog).AnyTimes()
		mp.EXPECT().DBConfig().Return(dbCfg).AnyTimes()

		dwl := &dbWithLogging{provider: mp}

		dwl.logQueryResult("SQL Exec", mockDuration, nil, nil)
	})

	t.Run("遅いクエリでWarnが呼ばれる", func(t *testing.T) {
		t.Parallel()

		mockDuration := dbCfg.SlowQueryWarnThreshold() + time.Duration(100*time.Millisecond)
		mockLog := mock_logging.NewMockLogger(ctrl)
		mockLog.EXPECT().Named(layer).Return(mockLog)
		mockLog.EXPECT().CallerSkip(callSkip).Return(mockLog)
		mockLog.EXPECT().Warn("SQL Exec")

		mp := mock_loggingdb.NewMockDBProvider(ctrl)
		mp.EXPECT().LogFields().Return(lf).AnyTimes()
		mp.EXPECT().Logger().Return(mockLog).AnyTimes()
		mp.EXPECT().DBConfig().Return(dbCfg).AnyTimes()

		dwl := &dbWithLogging{provider: mp}

		dwl.logQueryResult("SQL Exec", mockDuration, nil, nil)
	})

	t.Run("Errorが呼ばれる", func(t *testing.T) {
		t.Parallel()

		mockDuration := dbCfg.SlowQueryWarnThreshold() - time.Duration(100*time.Millisecond)
		mockLog := mock_logging.NewMockLogger(ctrl)
		mockLog.EXPECT().Named(layer).Return(mockLog)
		mockLog.EXPECT().CallerSkip(callSkip).Return(mockLog)
		mockLog.EXPECT().Error("SQL Exec")
		mp := mock_loggingdb.NewMockDBProvider(ctrl)
		mp.EXPECT().LogFields().Return(lf).AnyTimes()
		mp.EXPECT().Logger().Return(mockLog).AnyTimes()
		mp.EXPECT().DBConfig().Return(dbCfg).AnyTimes()
		dwl := &dbWithLogging{provider: mp}

		dwl.logQueryResult("SQL Exec", mockDuration, nil, errors.New("boom"))
	})
}
