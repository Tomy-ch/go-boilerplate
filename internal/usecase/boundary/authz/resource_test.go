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

		t.Run("生成後に入力ポインタの指す値を変更してもResourceの所有者IDは不変", func(t *testing.T) {
			t.Parallel()
			original := uuid.NewTestFromSalt(t, "owner")
			id := original

			r := NewResource("user", &id)

			id = uuid.NewTestFromSalt(t, "mutated")

			require.NotNil(t, r.OwnerID())
			assert.Equal(t, original, *r.OwnerID())
			assert.NotEqual(t, id, *r.OwnerID())
		})
	})
}

func TestResource_Kind(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("生成時に渡した種別を返す", func(t *testing.T) {
			t.Parallel()
			id := uuid.NewTestFromSalt(t, "owner")
			r := NewResource("user", &id)

			assert.Equal(t, "user", r.Kind())
		})
	})
}

func TestResource_OwnerID(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("生成時に渡した所有者IDと同値を返す", func(t *testing.T) {
			t.Parallel()
			id := uuid.NewTestFromSalt(t, "owner")
			r := NewResource("user", &id)

			require.NotNil(t, r.OwnerID())
			assert.Equal(t, id, *r.OwnerID())
		})

		t.Run("所有者概念がない場合はnilを返す", func(t *testing.T) {
			t.Parallel()
			r := NewResource("user", nil)

			assert.Nil(t, r.OwnerID())
		})

		t.Run("呼び出し毎に独立したポインタを返す", func(t *testing.T) {
			t.Parallel()
			id := uuid.NewTestFromSalt(t, "owner")
			r := NewResource("user", &id)

			first := r.OwnerID()
			second := r.OwnerID()

			require.NotNil(t, first)
			require.NotNil(t, second)
			assert.NotSame(t, first, second)
			assert.Equal(t, *first, *second)
		})

		t.Run("返り値のポインタの指す値を変更してもResourceの所有者IDは不変", func(t *testing.T) {
			t.Parallel()
			id := uuid.NewTestFromSalt(t, "owner")
			r := NewResource("user", &id)

			got := r.OwnerID()
			require.NotNil(t, got)
			*got = uuid.NewTestFromSalt(t, "mutated")

			require.NotNil(t, r.OwnerID())
			assert.Equal(t, id, *r.OwnerID())
			assert.NotEqual(t, *got, *r.OwnerID())
		})
	})
}
