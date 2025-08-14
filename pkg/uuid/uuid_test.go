package uuid

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()
	uuid, err := New()
	require.NoError(t, err)
	require.NotEmpty(t, uuid.String())
}

func TestNewTestFromSalt(t *testing.T) {
	t.Parallel()
	salt := "test-salt"
	uuid1 := NewTestFromSalt(t, salt)
	uuid2 := NewTestFromSalt(t, salt)
	require.Equal(t, uuid1, uuid2)
}

func TestString(t *testing.T) {
	t.Parallel()
	uuid, err := New()
	require.NoError(t, err)
	require.NotEmpty(t, uuid.String())
}

func TestEqual(t *testing.T) {
	t.Parallel()
	uuid1, err := New()
	require.NoError(t, err)
	uuid2 := uuid1
	require.True(t, uuid1.Equal(uuid2))
}

func TestToPtr(t *testing.T) {
	t.Parallel()
	uuid, err := New()
	require.NoError(t, err)
	require.NotNil(t, uuid.ToPtr())
}

func TestEqualPtr(t *testing.T) {
	t.Parallel()
	uuid1, err := New()
	require.NoError(t, err)
	uuid2 := uuid1.ToPtr()
	require.True(t, uuid1.EqualPtr(uuid2))
}

func TestParse(t *testing.T) {
	t.Parallel()
	uuid, err := New()
	require.NoError(t, err)

	t.Run("正常系/問題なく解析ができると、uuidが得られる", func(t *testing.T) {
		t.Parallel()
		actual, err := Parse(uuid.String())
		require.NoError(t, err)
		require.True(t, uuid.Equal(actual))
	})

	t.Run("異常系/無効な文字列を解析するとエラーが返る", func(t *testing.T) {
		t.Parallel()
		actual, err := Parse("invalid-uuid")
		require.Empty(t, actual)
		require.Error(t, err)
	})
}
