package fixcollation

import (
	"context"
	"errors"
	"testing"

	"go-boilerplate/internal/logging"
	mock_exec "go-boilerplate/pkg/exec/mock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestValidateDatabaseName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "正常系_localは許可", input: "local", wantErr: false},
		{name: "正常系_testは許可", input: "test", wantErr: false},
		{name: "異常系_空文字は不許可", input: "", wantErr: true},
		{name: "異常系_想定外のDB名は不許可", input: "production", wantErr: true},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateDatabaseName(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestFixCollation(t *testing.T) {
	t.Parallel()

	t.Run("正常系_REINDEXとALTERを順に実行する", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		runner := mock_exec.NewMockRunner(ctrl)

		// 2 文（REINDEX → ALTER）が psql 経由で実行されること。
		runner.EXPECT().Output(gomock.Any(), workDir, psqlCommand, gomock.Any()).Return(nil, nil).Times(2)

		err := fixCollation(context.Background(), runner, logging.NewTestLogger(t), "postgres://dsn", "local")
		require.NoError(t, err)
	})

	t.Run("異常系_psql失敗時はエラーを返し後続を実行しない", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		runner := mock_exec.NewMockRunner(ctrl)

		// 1 文目で失敗 → 2 文目は実行されない（Times(1)）。
		runner.EXPECT().Output(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("psql failed")).Times(1)

		err := fixCollation(context.Background(), runner, logging.NewTestLogger(t), "postgres://dsn", "local")
		require.Error(t, err)
	})
}

func TestRunFix(t *testing.T) {
	t.Parallel()

	t.Run("正常系_検証通過後にDSNを解決しcollation修正を実行する", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		runner := mock_exec.NewMockRunner(ctrl)
		runner.EXPECT().Output(gomock.Any(), workDir, psqlCommand, gomock.Any()).Return(nil, nil).Times(2)

		loadDSN := func() (string, error) { return "postgres://dsn", nil }
		err := RunFix(context.Background(), runner, logging.NewTestLogger(t), "local", loadDSN)
		require.NoError(t, err)
	})

	t.Run("異常系_不正なDB名はDSN解決前に弾く", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		runner := mock_exec.NewMockRunner(ctrl)

		called := false
		loadDSN := func() (string, error) {
			called = true
			return "", nil
		}
		err := RunFix(context.Background(), runner, logging.NewTestLogger(t), "production", loadDSN)
		require.Error(t, err)
		assert.False(t, called)
	})

	t.Run("異常系_DSN解決に失敗すると修正せずエラー", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		runner := mock_exec.NewMockRunner(ctrl)

		loadDSN := func() (string, error) { return "", errors.New("config failed") }
		err := RunFix(context.Background(), runner, logging.NewTestLogger(t), "local", loadDSN)
		require.Error(t, err)
	})
}
