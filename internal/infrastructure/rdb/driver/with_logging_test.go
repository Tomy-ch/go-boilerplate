package rdbdriver

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/stretchr/testify/require"
)

type mockDBTX struct{}

func (m *mockDBTX) ExecContext(_ context.Context, _ string, _ ...any) (sql.Result, error) {
	return nil, errors.New("not implemented")
}

func (m *mockDBTX) PrepareContext(_ context.Context, _ string) (*sql.Stmt, error) {
	return nil, errors.New("not implemented")
}

func (m *mockDBTX) QueryContext(_ context.Context, _ string, _ ...any) (*sql.Rows, error) {
	return nil, errors.New("not implemented")
}

func (m *mockDBTX) QueryRowContext(_ context.Context, _ string, _ ...any) *sql.Row {
	return nil
}

func TestNewLoggingDB(t *testing.T) {
	t.Parallel()

	t.Run("DBTXをラップしてログ出力機能を追加する", func(t *testing.T) {
		mockDB := &mockDBTX{}
		logger := zap.NewNop()

		wrappedDB := NewLoggingDB(mockDB, logger)

		require.NotNil(t, wrappedDB)
	})
}

func TestBuildZapFields(t *testing.T) {
	t.Parallel()

	t.Run("クエリと実行時間を含むzap.Fieldを構築する", func(t *testing.T) {
		query := "SELECT * FROM users"
		expectedDuration := time.Duration(100)
		duration := expectedDuration * time.Millisecond
		sec := float64(duration) / float64(time.Second)

		fields := buildZapFields(query, duration)

		require.Len(t, fields, 2)
		require.Equal(t, zap.String("query", query), fields[0])
		require.Equal(t, zap.Float64("dur_sec", sec), fields[1])
	})
}

func TestBuildZapWithArgsFields(t *testing.T) {
	t.Parallel()

	t.Run("クエリ、実行時間、引数を含むzap.Fieldを構築する", func(t *testing.T) {
		query := "SELECT * FROM users WHERE id = ?"
		expectedDuration := time.Duration(200)
		duration := expectedDuration * time.Millisecond
		args := []any{1}
		sec := float64(duration) / float64(time.Second)

		fields := buildZapWithArgsFields(query, duration, args...)

		require.Len(t, fields, 3)
		require.Equal(t, zap.String("query", query), fields[0])
		require.Equal(t, zap.Float64("dur_sec", sec), fields[1])
		require.Equal(t, zap.Any("args", args), fields[2])
	})
}
