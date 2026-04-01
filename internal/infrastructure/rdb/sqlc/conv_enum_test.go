package sqlc

import (
	"testing"

	"boilerplate-go/internal/infrastructure/rdb/sqlc/gen"

	"github.com/stretchr/testify/require"
)

func TestBoolToActiveState(t *testing.T) {
	t.Parallel()

	t.Run("true", func(t *testing.T) {
		t.Parallel()
		b := true

		expected := gen.ActiveStateActive
		actual := BoolToActiveState(b)

		require.Equal(t, expected, actual)
	})

	t.Run("false", func(t *testing.T) {
		t.Parallel()
		b := false

		expected := gen.ActiveStateDeleted
		actual := BoolToActiveState(b)

		require.Equal(t, expected, actual)
	})
}
