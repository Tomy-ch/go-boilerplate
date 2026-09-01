package inquirymessage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAuthorKind(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("利用者を表す文字列から種別を生成する", func(t *testing.T) {
			t.Parallel()

			kind, err := NewAuthorKind("user")
			require.NoError(t, err)
			assert.Equal(t, AuthorKindUser, kind)
		})

		t.Run("回答者を表す文字列から種別を生成する", func(t *testing.T) {
			t.Parallel()

			kind, err := NewAuthorKind("operator")
			require.NoError(t, err)
			assert.Equal(t, AuthorKindOperator, kind)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("既知の2値以外はErrInvalidAuthorKindを返す", func(t *testing.T) {
			t.Parallel()
			_, err := NewAuthorKind("admin")
			require.ErrorIs(t, err, ErrInvalidAuthorKind)
		})

		t.Run("空文字はErrInvalidAuthorKindを返す", func(t *testing.T) {
			t.Parallel()
			_, err := NewAuthorKind("")
			require.ErrorIs(t, err, ErrInvalidAuthorKind)
		})
	})
}

func TestAuthorKind_valid(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("利用者は既知の種別として扱う", func(t *testing.T) {
			t.Parallel()
			assert.True(t, AuthorKindUser.valid())
		})

		t.Run("回答者は既知の種別として扱う", func(t *testing.T) {
			t.Parallel()
			assert.True(t, AuthorKindOperator.valid())
		})

		t.Run("生成関数を経ていないゼロ値は既知の種別として扱わない", func(t *testing.T) {
			t.Parallel()
			assert.False(t, AuthorKind("").valid())
		})

		t.Run("既知の2値以外の文字列は既知の種別として扱わない", func(t *testing.T) {
			t.Parallel()
			assert.False(t, AuthorKind("admin").valid())
		})
	})
}

func TestAuthorKind_String(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("利用者はuserを返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "user", AuthorKindUser.String())
		})

		t.Run("回答者はoperatorを返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "operator", AuthorKindOperator.String())
		})
	})
}
