package loggingdb

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/infrastructure/rdb/driver"
	mock_loggingdb "boilerplate-go/internal/infrastructure/rdb/driver/loggingdb/mock"
	mock_driver "boilerplate-go/internal/infrastructure/rdb/driver/mock"
	"boilerplate-go/internal/logging"
	mock_logging "boilerplate-go/internal/logging/mock"
	"boilerplate-go/internal/observability"

	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

func Test_dbWithLogging_ExecContext(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	lf := logging.NewTestLogFieldBuilder(t)
	lg := logging.NewTestLogger(t)
	noopLayerTracer := observability.NewNoopLayerTracer(t)

	ctrl := gomock.NewController(t)

	md := mock_driver.NewMockDBTX(ctrl)

	var dbtx driver.DBTX = md

	md.EXPECT().ExecContext(gomock.Any(), "INSERT INTO users (name) VALUES (?)", "alice").Return(nil, nil)

	mp := mock_loggingdb.NewMockDBProvider(ctrl)
	mp.EXPECT().LogFields().Return(lf).AnyTimes()
	mp.EXPECT().Logger().Return(lg).AnyTimes()
	mp.EXPECT().DBConfig().Return(dbCfg).AnyTimes()
	mp.EXPECT().LayerTracer().Return(noopLayerTracer).AnyTimes()

	dwl := &dbWithLogging{db: dbtx, ctx: context.Background(), provider: mp}

	res, err := dwl.ExecContext(context.Background(), "INSERT INTO users (name) VALUES (?)", "alice")
	require.NoError(t, err)

	require.Nil(t, res)
}

func Test_dbWithLogging_PrepareContext(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	lf := logging.NewTestLogFieldBuilder(t)
	lg := logging.NewTestLogger(t)
	noopLayerTracer := observability.NewNoopLayerTracer(t)

	ctrl := gomock.NewController(t)

	md := mock_driver.NewMockDBTX(ctrl)
	var dbtx driver.DBTX = md
	md.EXPECT().PrepareContext(gomock.Any(), "INSERT INTO users (name) VALUES (?)").Return((*sql.Stmt)(nil), nil)

	mp := mock_loggingdb.NewMockDBProvider(ctrl)
	mp.EXPECT().LogFields().Return(lf).AnyTimes()
	mp.EXPECT().Logger().Return(lg).AnyTimes()
	mp.EXPECT().DBConfig().Return(dbCfg).AnyTimes()
	mp.EXPECT().LayerTracer().Return(noopLayerTracer).AnyTimes()

	dwl := &dbWithLogging{db: dbtx, ctx: context.Background(), provider: mp}

	stmt, err := dwl.PrepareContext(context.Background(), "INSERT INTO users (name) VALUES (?)")
	require.NoError(t, err)
	require.Nil(t, stmt)
}

func Test_dbWithLogging_QueryContext(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	lf := logging.NewTestLogFieldBuilder(t)
	lg := logging.NewTestLogger(t)
	noopLayerTracer := observability.NewNoopLayerTracer(t)

	ctrl := gomock.NewController(t)

	md := mock_driver.NewMockDBTX(ctrl)
	var dbtx driver.DBTX = md
	md.EXPECT().QueryContext(gomock.Any(), "select 1").Return((*sql.Rows)(nil), nil)

	mp := mock_loggingdb.NewMockDBProvider(ctrl)
	mp.EXPECT().LogFields().Return(lf).AnyTimes()
	mp.EXPECT().Logger().Return(lg).AnyTimes()
	mp.EXPECT().DBConfig().Return(dbCfg).AnyTimes()
	mp.EXPECT().LayerTracer().Return(noopLayerTracer).AnyTimes()

	dwl := &dbWithLogging{db: dbtx, ctx: context.Background(), provider: mp}

	//nolint:rowserrcheck // このテストではrowsがnilであることを確認するだけなのでCloseは不要
	rows, err := dwl.QueryContext(context.Background(), "select 1")
	require.NoError(t, err)
	require.Nil(t, rows)
}

func Test_dbWithLogging_QueryRowContext(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	lf := logging.NewTestLogFieldBuilder(t)
	lg := logging.NewTestLogger(t)
	noopLayerTracer := observability.NewNoopLayerTracer(t)

	ctrl := gomock.NewController(t)

	md := mock_driver.NewMockDBTX(ctrl)
	var dbtx driver.DBTX = md
	md.EXPECT().QueryRowContext(gomock.Any(), "SELECT ?", 1).Return((*sql.Row)(nil))

	mp := mock_loggingdb.NewMockDBProvider(ctrl)
	mp.EXPECT().LogFields().Return(lf).AnyTimes()
	mp.EXPECT().Logger().Return(lg).AnyTimes()
	mp.EXPECT().DBConfig().Return(dbCfg).AnyTimes()
	mp.EXPECT().LayerTracer().Return(noopLayerTracer).AnyTimes()

	dwl := &dbWithLogging{db: dbtx, ctx: context.Background(), provider: mp}

	row := dwl.QueryRowContext(context.Background(), "SELECT ?", 1)
	require.Nil(t, row)
}

func Test_dbWithLogging_buildSQLStartLogFields(t *testing.T) {
	t.Parallel()

	lf := logging.NewTestLogFieldBuilder(t)

	tc := observability.TraceContext{}
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

	lf := logging.NewTestLogFieldBuilder(t)

	tc := observability.TraceContext{}
	funcName := "TestBuildSQLLogFields"
	query := "select 1"
	expectedDuration := time.Duration(100 * time.Millisecond)

	dwl := &dbWithLogging{
		provider: &provider{
			lf: lf,
		},
	}

	actual := dwl.buildSQLEndLogFields(tc, funcName, query, expectedDuration, nil, nil)
	require.NotEmpty(t, actual)
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
