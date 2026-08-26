package uuid

import (
	"encoding/json"
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

	var zero UUID
	assert.True(t, zero.EqualPtr(zero.ToPtr()))
}

func TestParse(t *testing.T) {
	t.Parallel()
	uuid, err := New()
	require.NoError(t, err)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("問題なく解析ができると、uuidが得られる", func(t *testing.T) {
			t.Parallel()
			actual, err := Parse(uuid.String())
			require.NoError(t, err)
			assert.True(t, uuid.Equal(actual))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("無効な文字列を解析するとエラーが返る", func(t *testing.T) {
			t.Parallel()
			actual, err := Parse("invalid-uuid")
			assert.Empty(t, actual)
			require.Error(t, err)
		})
	})
}

func TestUUID_MarshalJSON(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("JSON 文字列として符号化する", func(t *testing.T) {
			t.Parallel()
			u, err := Parse("b1d4e0f2-3c5a-4b6d-8e7f-1a2b3c4d5e6f")
			require.NoError(t, err)

			b, err := json.Marshal(u)
			require.NoError(t, err)
			assert.JSONEq(t, `"b1d4e0f2-3c5a-4b6d-8e7f-1a2b3c4d5e6f"`, string(b))
		})

		t.Run("ゼロ値もゼロUUIDの文字列として符号化する", func(t *testing.T) {
			t.Parallel()
			b, err := json.Marshal(UUID{})
			require.NoError(t, err)
			assert.JSONEq(t, `"00000000-0000-0000-0000-000000000000"`, string(b))
		})
	})
}

func TestUUID_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("JSON 文字列から復元する", func(t *testing.T) {
			t.Parallel()
			var u UUID
			require.NoError(t, json.Unmarshal([]byte(`"b1d4e0f2-3c5a-4b6d-8e7f-1a2b3c4d5e6f"`), &u))
			assert.Equal(t, "b1d4e0f2-3c5a-4b6d-8e7f-1a2b3c4d5e6f", u.String())
		})

		t.Run("符号化した値と往復しても同じUUIDに戻る", func(t *testing.T) {
			t.Parallel()
			want, err := New()
			require.NoError(t, err)
			b, err := json.Marshal(want)
			require.NoError(t, err)

			var got UUID
			require.NoError(t, json.Unmarshal(b, &got))
			assert.True(t, want.Equal(got))
		})

		t.Run("JSON null は値を変更しない", func(t *testing.T) {
			t.Parallel()
			want, err := New()
			require.NoError(t, err)

			got := want
			require.NoError(t, got.UnmarshalJSON([]byte("null")))
			assert.True(t, want.Equal(got))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("UUIDとして解析できない文字列はエラーを返し値を変更しない", func(t *testing.T) {
			t.Parallel()
			want, err := New()
			require.NoError(t, err)

			got := want
			require.Error(t, got.UnmarshalJSON([]byte(`"invalid-uuid"`)))
			assert.True(t, want.Equal(got))
		})

		t.Run("文字列以外の値はエラーを返し値を変更しない", func(t *testing.T) {
			t.Parallel()
			want, err := New()
			require.NoError(t, err)

			got := want
			require.Error(t, got.UnmarshalJSON([]byte(`{}`)))
			assert.True(t, want.Equal(got))
		})
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
