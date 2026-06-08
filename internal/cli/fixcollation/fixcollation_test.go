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

func TestNewCommand(t *testing.T) {
	t.Parallel()

	cmd := NewCommand()
	assert.Equal(t, "fix-collation", cmd.Use)
	f := cmd.Flags().Lookup("database")
	require.NotNil(t, f)
	assert.Equal(t, "local", f.DefValue)
}
