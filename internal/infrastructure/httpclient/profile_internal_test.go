package httpclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_staticRegistry_Profile(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("登録済みキーは対応するProfileを返す", func(t *testing.T) {
			t.Parallel()

			custom := DefaultProfile()
			custom.MaxAttempts = 99
			reg := NewRegistry(map[Downstream]Profile{"known": custom})

			got := reg.Profile("known")

			assert.Equal(t, 99, got.MaxAttempts)
		})

		t.Run("未登録キーはfallbackのDefaultProfileを返す", func(t *testing.T) {
			t.Parallel()

			reg := NewRegistry(map[Downstream]Profile{})

			got := reg.Profile("unknown")

			assert.Equal(t, DefaultProfile(), got)
		})
	})
}
