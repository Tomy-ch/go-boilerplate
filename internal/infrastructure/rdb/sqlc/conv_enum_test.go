package sqlc

import (
	"testing"

	"boilerplate-go/internal/infrastructure/rdb/sqlc/gen"

	"github.com/stretchr/testify/require"
)

func TestBoolPtrToDeletedState(t *testing.T) {
	t.Parallel()

	t.Run("nil", func(t *testing.T) {
		t.Parallel()

		expected := gen.DeletedStateAll
		actual := BoolPtrToDeletedState(nil)

		require.Equal(t, expected, actual)
	})

	t.Run("true", func(t *testing.T) {
		t.Parallel()
		b := true

		expected := gen.DeletedStateDeleted
		actual := BoolPtrToDeletedState(&b)

		require.Equal(t, expected, actual)
	})

	t.Run("false", func(t *testing.T) {
		t.Parallel()
		b := false

		expected := gen.DeletedStateActive
		actual := BoolPtrToDeletedState(&b)

		require.Equal(t, expected, actual)
	})
}

func TestBoolToDeletedState(t *testing.T) {
	t.Parallel()

	t.Run("true", func(t *testing.T) {
		t.Parallel()
		b := true

		expected := gen.DeletedStateDeleted
		actual := BoolToDeletedState(b)

		require.Equal(t, expected, actual)
	})

	t.Run("false", func(t *testing.T) {
		t.Parallel()
		b := false

		expected := gen.DeletedStateActive
		actual := BoolToDeletedState(b)

		require.Equal(t, expected, actual)
	})
}
