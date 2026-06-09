package basicauth

import (
	"testing"

	"go-boilerplate/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBasicAuthValidator(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	mtc := config.NewMetricsConfig(cfg)
	validator := NewBasicAuthValidator(mtc)

	cases := []struct {
		name     string
		username string
		password string
		wantOK   bool
	}{
		{"正常系_ユーザー名とパスワードが一致する場合", mtc.UserName(), mtc.Password(), true},
		{"異常系_両方とも不一致の場合", "wrong-user", "wrong-password", false},
		{"異常系_ユーザー名のみ一致しパスワードが不一致の場合", mtc.UserName(), "wrong-password", false},
		{"異常系_パスワードのみ一致しユーザー名が不一致の場合", "wrong-user", mtc.Password(), false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ok, err := validator(tt.username, tt.password, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.wantOK, ok)
		})
	}
}
