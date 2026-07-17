package uuid

import (
	"testing"

	googleuuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/pkg/xerrors"
)

// failingReader は、常に読み取り失敗する乱数源です（NewV7 の失敗分岐検証用）。
type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, xerrors.New("rand source failure") }

// グローバルな SetRand を差し替えるため直列実行する（他テストの New 呼び出しとの競合回避）。
//
//nolint:paralleltest // SetRand はプロセス共有のグローバル状態のため並列化不可
func TestNew(t *testing.T) {
	t.Run("正常系", func(t *testing.T) {
		t.Run("UUIDv7を生成し文字列表現が空でない", func(t *testing.T) {
			u, err := New()
			require.NoError(t, err)
			assert.NotEmpty(t, u.String())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("乱数源が失敗するとエラーを返す", func(t *testing.T) {
			googleuuid.SetRand(failingReader{})
			t.Cleanup(func() { googleuuid.SetRand(nil) })

			u, err := New()
			require.Error(t, err)
			assert.True(t, u.IsNil())
		})
	})
}

func TestNewTestFromSalt(t *testing.T) {
	t.Parallel()
	salt := "test-salt"
	uuid1 := NewTestFromSalt(t, salt)
	uuid2 := NewTestFromSalt(t, salt)
	assert.Equal(t, uuid1, uuid2)
	assert.NotEqual(t, uuid1, NewTestFromSalt(t, "other-salt"))
}

func TestUUID_Bytes(t *testing.T) {
	t.Parallel()
	uuid, err := New()
	require.NoError(t, err)
	bytes := uuid.Bytes()
	assert.Len(t, bytes, 16)
	assert.Equal(t, uuid.b, bytes)
}

func TestUUID_IsNil(t *testing.T) {
	t.Parallel()
	var nilUUID UUID
	assert.True(t, nilUUID.IsNil())

	uuid, err := New()
	require.NoError(t, err)
	assert.False(t, uuid.IsNil())
}

func TestUUID_ToPrimitive(t *testing.T) {
	t.Parallel()
	uuid, err := New()
	require.NoError(t, err)
	primitive := uuid.ToPrimitive()
	assert.Equal(t, toGoogle(uuid), primitive)
}

func TestFromPrimitive(t *testing.T) {
	t.Parallel()
	uuid, err := New()
	require.NoError(t, err)
	got := FromPrimitive(uuid.ToPrimitive())
	assert.Equal(t, uuid, got)
}

func TestUUID_String(t *testing.T) {
	t.Parallel()
	uuid, err := New()
	require.NoError(t, err)
	assert.NotEmpty(t, uuid.String())
}

func TestUUID_Equal(t *testing.T) {
	t.Parallel()
	uuid1, err := New()
	require.NoError(t, err)
	uuid2 := uuid1
	assert.True(t, uuid1.Equal(uuid2))

	uuid3, err := New()
	require.NoError(t, err)
	assert.False(t, uuid1.Equal(uuid3))
}

func TestUUID_ToPtr(t *testing.T) {
	t.Parallel()
	uuid, err := New()
	require.NoError(t, err)
	assert.NotNil(t, uuid.ToPtr())
}

func TestUUID_EqualPtr(t *testing.T) {
	t.Parallel()
	uuid1, err := New()
	require.NoError(t, err)
	uuid2 := uuid1.ToPtr()
	assert.True(t, uuid1.EqualPtr(uuid2))
	assert.False(t, uuid1.EqualPtr(nil))

	uuid3, err := New()
	require.NoError(t, err)
	assert.False(t, uuid1.EqualPtr(uuid3.ToPtr()))
}

func TestParse(t *testing.T) {
	t.Parallel()
	uuid, err := New()
	require.NoError(t, err)

	t.Run("正常系/問題なく解析ができると、uuidが得られる", func(t *testing.T) {
		t.Parallel()
		actual, err := Parse(uuid.String())
		require.NoError(t, err)
		assert.True(t, uuid.Equal(actual))
	})

	t.Run("異常系/無効な文字列を解析するとエラーが返る", func(t *testing.T) {
		t.Parallel()
		actual, err := Parse("invalid-uuid")
		assert.Empty(t, actual)
		require.Error(t, err)
	})
}

func TestUUID_Scan(t *testing.T) {
	t.Parallel()
	u, err := New()
	require.NoError(t, err)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("文字列からスキャンできる", func(t *testing.T) {
			t.Parallel()
			var s UUID
			err := s.Scan(u.String())
			require.NoError(t, err)
			assert.True(t, u.Equal(s))
		})

		t.Run("バイト列からスキャンできる", func(t *testing.T) {
			t.Parallel()
			var s UUID
			b := u.Bytes()
			err := s.Scan(b[:])
			require.NoError(t, err)
			assert.True(t, u.Equal(s))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("サポート外の型だとエラーになる", func(t *testing.T) {
			t.Parallel()
			var s UUID
			err := s.Scan(123)
			require.Error(t, err)
		})
	})
}

func TestUUID_Value(t *testing.T) {
	t.Parallel()
	u, err := New()
	require.NoError(t, err)

	v, err := u.Value()
	require.NoError(t, err)
	assert.Equal(t, u.String(), v)
}
