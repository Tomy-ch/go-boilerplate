package authz

import (
	"testing"

	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewResource(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("所有者IDありのResourceを生成する", func(t *testing.T) {
			t.Parallel()
			id := uuid.NewTestFromSalt(t, "owner")

			r := NewResource("user", &id)

			require.NotNil(t, r)
			assert.Equal(t, "user", r.Kind())
			require.NotNil(t, r.OwnerID())
			assert.Equal(t, id, *r.OwnerID())
		})

		t.Run("所有者IDなし_OwnerIDがnilのResourceを生成する", func(t *testing.T) {
			t.Parallel()

			r := NewResource("user", nil)

			require.NotNil(t, r)
			assert.Equal(t, "user", r.Kind())
			assert.Nil(t, r.OwnerID())
		})
	})
}
