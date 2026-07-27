package datetime

import (
	"testing"
	"time"

	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_toLocation(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("parse 成功時は同一瞬間を loc の時刻へ変換して返す", func(t *testing.T) {
			t.Parallel()
			jst, err := time.LoadLocation("Asia/Tokyo")
			require.NoError(t, err)
			base := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

			got, err := toLocation(jst, func() (time.Time, error) { return base, nil })
			require.NoError(t, err)
			assert.Equal(t, jst, got.Location())
			assert.True(t, got.Equal(base))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("loc が nil の場合はエラーを返し parse を評価しても結果は捨てる", func(t *testing.T) {
			t.Parallel()
			got, err := toLocation(nil, func() (time.Time, error) { return time.Now(), nil })
			require.ErrorIs(t, err, ErrNilLocation)
			assert.True(t, got.IsZero())
		})

		t.Run("parse がエラーを返す場合はそのエラーをそのまま伝播する", func(t *testing.T) {
			t.Parallel()
			jst, err := time.LoadLocation("Asia/Tokyo")
			require.NoError(t, err)
			sentinel := xerrors.New("parse failed")

			got, err := toLocation(jst, func() (time.Time, error) { return time.Time{}, sentinel })
			require.ErrorIs(t, err, sentinel)
			assert.True(t, got.IsZero())
		})
	})
}
