package uuid

import (
	"testing"

	guid "github.com/google/uuid"
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

func TestBytes(t *testing.T) {
	t.Parallel()
	uuid, err := New()
	require.NoError(t, err)
	bytes := uuid.Bytes()
	require.Len(t, bytes, 16)
	require.Equal(t, uuid.b, bytes)
}

func TestIsNil(t *testing.T) {
	t.Parallel()
	var nilUUID UUID
	require.True(t, nilUUID.IsNil())

	uuid, err := New()
	require.NoError(t, err)
	require.False(t, uuid.IsNil())
}

func TestToPrimitive(t *testing.T) {
	t.Parallel()
	uuid, err := New()
	require.NoError(t, err)
	primitive := uuid.ToPrimitive()
	require.Equal(t, toGoogle(uuid), primitive)
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

func TestFromPrimitive(t *testing.T) {
	t.Parallel()

	uuid, err := New()
	require.NoError(t, err)

	primitive := toGoogle(uuid)
	actual := FromPrimitive(primitive)
	require.True(t, uuid.Equal(actual))
}

func TestFromPrimitiveList(t *testing.T) {
	t.Parallel()
	uuid1, err := New()
	require.NoError(t, err)
	uuid2, err := New()
	require.NoError(t, err)

	primitiveList := []guid.UUID{
		toGoogle(uuid1),
		toGoogle(uuid2),
	}

	uuidList := FromPrimitiveList(primitiveList)
	require.Len(t, uuidList, 2)
	require.True(t, uuid1.Equal(uuidList[0]))
	require.True(t, uuid2.Equal(uuidList[1]))
}

func TestToPrimitiveList(t *testing.T) {
	t.Parallel()
	uuid1, err := New()
	require.NoError(t, err)
	uuid2, err := New()
	require.NoError(t, err)

	uuidList := []UUID{uuid1, uuid2}

	primitiveList := ToPrimitiveList(uuidList)
	require.Len(t, primitiveList, 2)
	require.Equal(t, toGoogle(uuid1), primitiveList[0])
	require.Equal(t, toGoogle(uuid2), primitiveList[1])
}

func TestToPrimitiveUniqueList(t *testing.T) {
	t.Parallel()
	uuid1, err := New()
	require.NoError(t, err)
	uuid2, err := New()
	require.NoError(t, err)

	uuidList := []UUID{uuid1, uuid2, uuid1, uuid2}

	primitiveList := ToPrimitiveUniqueList(uuidList)
	require.Len(t, primitiveList, 2)
	require.Contains(t, primitiveList, toGoogle(uuid1))
	require.Contains(t, primitiveList, toGoogle(uuid2))
}
