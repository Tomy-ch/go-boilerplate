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

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ユーザー名とパスワードが一致する場合、trueを返す", func(t *testing.T) {
			t.Parallel()
			ok, err := validator(mtc.UserName(), mtc.Password(), nil)
			require.NoError(t, err)
			assert.True(t, ok)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("両方とも不一致の場合、falseを返す", func(t *testing.T) {
			t.Parallel()
			ok, err := validator("wrong-user", "wrong-password", nil)
			require.NoError(t, err)
			assert.False(t, ok)
		})

		t.Run("ユーザー名のみ一致しパスワードが不一致の場合、falseを返す", func(t *testing.T) {
			t.Parallel()
			ok, err := validator(mtc.UserName(), "wrong-password", nil)
			require.NoError(t, err)
			assert.False(t, ok)
		})

		t.Run("パスワードのみ一致しユーザー名が不一致の場合、falseを返す", func(t *testing.T) {
			t.Parallel()
			ok, err := validator("wrong-user", mtc.Password(), nil)
			require.NoError(t, err)
			assert.False(t, ok)
		})
	})
}
