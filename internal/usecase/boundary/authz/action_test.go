package authz

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAction_String(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ActionUserGetの文字列表現を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "user:get", ActionUserGet.String())
		})

		t.Run("ActionUserUpdateの文字列表現を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "user:update", ActionUserUpdate.String())
		})

		t.Run("ActionUserDeleteの文字列表現を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "user:delete", ActionUserDelete.String())
		})

		t.Run("ActionProductCreateの文字列表現を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "product:create", ActionProductCreate.String())
		})

		t.Run("ActionPurchaseShipの文字列表現を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "purchase:ship", ActionPurchaseShip.String())
		})
	})
}
