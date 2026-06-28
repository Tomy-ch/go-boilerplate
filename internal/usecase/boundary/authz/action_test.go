package authz

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAction_String(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Actionの文字列表現を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "user:delete", ActionUserDelete.String())
		})
	})
}
