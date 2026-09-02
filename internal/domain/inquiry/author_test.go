package inquiry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"
)

func newTestAuthor(t *testing.T) Author {
	t.Helper()
	author, err := NewAuthor(AuthorKindUser, uuidtestkit.NewTestFromSalt(t, "subject"))
	require.NoError(t, err)
	return author
}

func TestNewAuthor(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("利用者の種別と主体から送り手を生成する", func(t *testing.T) {
			t.Parallel()
			subjectID := uuidtestkit.NewTestFromSalt(t, "subject")

			author, err := NewAuthor(AuthorKindUser, subjectID)
			require.NoError(t, err)
			assert.Equal(t, AuthorKindUser, author.Kind())
			assert.Equal(t, subjectID, author.SubjectID())
		})

		t.Run("回答者の種別と主体から送り手を生成する", func(t *testing.T) {
			t.Parallel()
			subjectID := uuidtestkit.NewTestFromSalt(t, "operator")

			author, err := NewAuthor(AuthorKindOperator, subjectID)
			require.NoError(t, err)
			assert.Equal(t, AuthorKindOperator, author.Kind())
			assert.Equal(t, subjectID, author.SubjectID())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("種別が既知の2値でなければErrInvalidAuthorKindを返す", func(t *testing.T) {
			t.Parallel()
			_, err := NewAuthor(AuthorKind("admin"), uuidtestkit.NewTestFromSalt(t, "subject"))
			require.ErrorIs(t, err, ErrInvalidAuthorKind)
		})

		t.Run("主体が未設定ならErrInvalidAuthorSubjectを返す", func(t *testing.T) {
			t.Parallel()
			_, err := NewAuthor(AuthorKindUser, uuid.UUID{})
			require.ErrorIs(t, err, ErrInvalidAuthorSubject)
		})
	})
}

func Test_validateAuthor(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("既知の種別と設定済みの主体を受け入れる", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, validateAuthor(AuthorKindOperator, uuidtestkit.NewTestFromSalt(t, "subject")))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("種別が未設定ならErrInvalidAuthorKindを返す", func(t *testing.T) {
			t.Parallel()
			err := validateAuthor(AuthorKind(""), uuidtestkit.NewTestFromSalt(t, "subject"))
			require.ErrorIs(t, err, ErrInvalidAuthorKind)
		})

		t.Run("主体が未設定ならErrInvalidAuthorSubjectを返す", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, validateAuthor(AuthorKindUser, uuid.UUID{}), ErrInvalidAuthorSubject)
		})

		t.Run("種別と主体の双方が不正なら種別のエラーを返す", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, validateAuthor(AuthorKind("admin"), uuid.UUID{}), ErrInvalidAuthorKind)
		})
	})
}

func TestAuthor_Kind(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("生成に用いた種別を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, AuthorKindUser, newTestAuthor(t).Kind())
		})
	})
}

func TestAuthor_SubjectID(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("生成に用いた主体を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, uuidtestkit.NewTestFromSalt(t, "subject"), newTestAuthor(t).SubjectID())
		})
	})
}
