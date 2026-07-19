package prefecture

import (
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_rowToPrefecture(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("有効な行からエンティティを再構築する", func(t *testing.T) {
			t.Parallel()

			id, err := uuid.New()
			require.NoError(t, err)

			entity, err := rowToPrefecture(id, "東京都", int16(13))
			require.NoError(t, err)
			require.NotNil(t, entity)
			assert.Equal(t, id, entity.ID())
			assert.Equal(t, "東京都", entity.Name())
			assert.Equal(t, 13, entity.Code())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("再構築時の検証失敗はErrInternalへ正規化され元の分類は露出しない", func(t *testing.T) {
			t.Parallel()

			id, err := uuid.New()
			require.NoError(t, err)

			// code=0 は JIS 範囲(1〜47)外のため domain 構築が失敗する。
			entity, err := rowToPrefecture(id, "東京都", int16(0))
			require.Error(t, err)
			require.Nil(t, entity)
			require.ErrorIs(t, err, apperror.ErrInternal)
		})
	})
}
