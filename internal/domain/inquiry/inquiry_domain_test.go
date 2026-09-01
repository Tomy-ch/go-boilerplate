package inquiry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"
)

func newTestInquiry(t *testing.T) *Inquiry {
	t.Helper()
	i, err := New(uuidtestkit.NewTestFromSalt(t, "inquiry"), Attributes{UserID: uuidtestkit.NewTestFromSalt(t, "user")})
	require.NoError(t, err)
	return i
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("利用者を指定すると生成できる", func(t *testing.T) {
			t.Parallel()
			id, userID := uuidtestkit.NewTestFromSalt(t, "inquiry"), uuidtestkit.NewTestFromSalt(t, "user")

			i, err := New(id, Attributes{UserID: userID})
			require.NoError(t, err)
			assert.Equal(t, id, i.ID())
			assert.Equal(t, userID, i.UserID())
		})

		t.Run("生成直後は作成日時と更新日時がゼロ値になる", func(t *testing.T) {
			t.Parallel()

			i, err := New(uuidtestkit.NewTestFromSalt(t, "inquiry"), Attributes{UserID: uuidtestkit.NewTestFromSalt(t, "user")})
			require.NoError(t, err)
			assert.True(t, i.CreatedAt().IsZero())
			assert.True(t, i.UpdatedAt().IsZero())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("IDが未設定ならErrInvalidIDを返す", func(t *testing.T) {
			t.Parallel()
			_, err := New(uuid.UUID{}, Attributes{UserID: uuidtestkit.NewTestFromSalt(t, "id")})
			require.ErrorIs(t, err, ErrInvalidID)
		})

		t.Run("利用者が未設定ならErrInvalidUserIDを返す", func(t *testing.T) {
			t.Parallel()
			_, err := New(uuidtestkit.NewTestFromSalt(t, "id"), Attributes{})
			require.ErrorIs(t, err, ErrInvalidUserID)
		})
	})
}

func TestReconstruct(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("作成日時と更新日時を含めて再構築する", func(t *testing.T) {
			t.Parallel()
			id, userID := uuidtestkit.NewTestFromSalt(t, "inquiry"), uuidtestkit.NewTestFromSalt(t, "user")
			createdAt := time.Date(2026, time.September, 1, 10, 0, 0, 0, time.UTC)
			updatedAt := createdAt.Add(time.Hour)

			i, err := Reconstruct(id, Attributes{UserID: userID, CreatedAt: createdAt, UpdatedAt: updatedAt})
			require.NoError(t, err)
			assert.Equal(t, id, i.ID())
			assert.Equal(t, userID, i.UserID())
			assert.Equal(t, createdAt, i.CreatedAt())
			assert.Equal(t, updatedAt, i.UpdatedAt())
		})

		t.Run("更新日時が作成日時と同時刻でも再構築できる", func(t *testing.T) {
			t.Parallel()
			at := time.Date(2026, time.September, 1, 10, 0, 0, 0, time.UTC)

			_, err := Reconstruct(
				uuidtestkit.NewTestFromSalt(t, "id"),
				Attributes{UserID: uuidtestkit.NewTestFromSalt(t, "id"), CreatedAt: at, UpdatedAt: at},
			)
			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("IDが未設定ならErrInvalidIDを返す", func(t *testing.T) {
			t.Parallel()
			_, err := Reconstruct(uuid.UUID{}, Attributes{UserID: uuidtestkit.NewTestFromSalt(t, "id")})
			require.ErrorIs(t, err, ErrInvalidID)
		})

		t.Run("更新日時が作成日時より前ならErrInvalidTimeを返す", func(t *testing.T) {
			t.Parallel()
			createdAt := time.Date(2026, time.September, 1, 10, 0, 0, 0, time.UTC)

			_, err := Reconstruct(uuidtestkit.NewTestFromSalt(t, "id"), Attributes{
				UserID:    uuidtestkit.NewTestFromSalt(t, "id"),
				CreatedAt: createdAt,
				UpdatedAt: createdAt.Add(-time.Nanosecond),
			})
			require.ErrorIs(t, err, ErrInvalidTime)
		})
	})
}

func Test_newInquiry(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("2つの入口が共有する検証を通ると属性を保持した集約を返す", func(t *testing.T) {
			t.Parallel()
			id, userID := uuidtestkit.NewTestFromSalt(t, "inquiry"), uuidtestkit.NewTestFromSalt(t, "user")
			at := time.Date(2026, time.September, 1, 10, 0, 0, 0, time.UTC)

			i, err := newInquiry(id, Attributes{UserID: userID, CreatedAt: at, UpdatedAt: at})
			require.NoError(t, err)
			assert.Equal(t, id, i.ID())
			assert.Equal(t, userID, i.UserID())
			assert.Equal(t, at, i.CreatedAt())
			assert.Equal(t, at, i.UpdatedAt())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("IDが未設定ならErrInvalidIDを返す", func(t *testing.T) {
			t.Parallel()
			_, err := newInquiry(uuid.UUID{}, Attributes{UserID: uuidtestkit.NewTestFromSalt(t, "id")})
			require.ErrorIs(t, err, ErrInvalidID)
		})

		t.Run("利用者が未設定ならErrInvalidUserIDを返す", func(t *testing.T) {
			t.Parallel()
			_, err := newInquiry(uuidtestkit.NewTestFromSalt(t, "id"), Attributes{})
			require.ErrorIs(t, err, ErrInvalidUserID)
		})
	})
}

func TestInquiry_ID(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("生成に用いたIDを返す", func(t *testing.T) {
			t.Parallel()
			id := uuidtestkit.NewTestFromSalt(t, "inquiry")

			i, err := New(id, Attributes{UserID: uuidtestkit.NewTestFromSalt(t, "id")})
			require.NoError(t, err)
			assert.Equal(t, id, i.ID())
		})
	})
}

func TestInquiry_UserID(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("生成に用いた利用者を返す", func(t *testing.T) {
			t.Parallel()
			userID := uuidtestkit.NewTestFromSalt(t, "user")

			i, err := New(uuidtestkit.NewTestFromSalt(t, "id"), Attributes{UserID: userID})
			require.NoError(t, err)
			assert.Equal(t, userID, i.UserID())
		})
	})
}

func TestInquiry_CreatedAt(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("生成直後はゼロ値を返す", func(t *testing.T) {
			t.Parallel()
			assert.True(t, newTestInquiry(t).CreatedAt().IsZero())
		})

		t.Run("再構築時に設定した作成日時を返す", func(t *testing.T) {
			t.Parallel()
			at := time.Date(2026, time.September, 1, 10, 0, 0, 0, time.UTC)

			i, err := Reconstruct(
				uuidtestkit.NewTestFromSalt(t, "id"),
				Attributes{UserID: uuidtestkit.NewTestFromSalt(t, "id"), CreatedAt: at, UpdatedAt: at},
			)
			require.NoError(t, err)
			assert.Equal(t, at, i.CreatedAt())
		})
	})
}

func TestInquiry_UpdatedAt(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("生成直後はゼロ値を返す", func(t *testing.T) {
			t.Parallel()
			assert.True(t, newTestInquiry(t).UpdatedAt().IsZero())
		})

		t.Run("再構築時に設定した更新日時を返す", func(t *testing.T) {
			t.Parallel()
			createdAt := time.Date(2026, time.September, 1, 10, 0, 0, 0, time.UTC)
			updatedAt := createdAt.Add(time.Hour)

			i, err := Reconstruct(uuidtestkit.NewTestFromSalt(t, "id"), Attributes{
				UserID: uuidtestkit.NewTestFromSalt(t, "id"), CreatedAt: createdAt, UpdatedAt: updatedAt,
			})
			require.NoError(t, err)
			assert.Equal(t, updatedAt, i.UpdatedAt())
		})
	})
}

func TestInquiry_Touch(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("現在の更新日時より後の時刻を渡すと更新日時が進む", func(t *testing.T) {
			t.Parallel()
			createdAt := time.Date(2026, time.September, 1, 10, 0, 0, 0, time.UTC)
			i, err := Reconstruct(uuidtestkit.NewTestFromSalt(t, "id"), Attributes{
				UserID: uuidtestkit.NewTestFromSalt(t, "id"), CreatedAt: createdAt, UpdatedAt: createdAt,
			})
			require.NoError(t, err)
			now := createdAt.Add(time.Hour)

			require.NoError(t, i.Touch(now))
			assert.Equal(t, now, i.UpdatedAt())
		})

		t.Run("現在の更新日時と同時刻でも成功する", func(t *testing.T) {
			t.Parallel()
			at := time.Date(2026, time.September, 1, 10, 0, 0, 0, time.UTC)
			i, err := Reconstruct(
				uuidtestkit.NewTestFromSalt(t, "id"),
				Attributes{UserID: uuidtestkit.NewTestFromSalt(t, "id"), CreatedAt: at, UpdatedAt: at},
			)
			require.NoError(t, err)

			require.NoError(t, i.Touch(at))
			assert.Equal(t, at, i.UpdatedAt())
		})

		t.Run("生成直後の集約はどの時刻でも成功する", func(t *testing.T) {
			t.Parallel()
			i := newTestInquiry(t)
			now := time.Date(2026, time.September, 1, 10, 0, 0, 0, time.UTC)

			require.NoError(t, i.Touch(now))
			assert.Equal(t, now, i.UpdatedAt())
		})

		t.Run("連続して呼ぶと更新日時は単調に進む", func(t *testing.T) {
			t.Parallel()
			i := newTestInquiry(t)
			first := time.Date(2026, time.September, 1, 10, 0, 0, 0, time.UTC)
			second := first.Add(time.Minute)

			require.NoError(t, i.Touch(first))
			require.NoError(t, i.Touch(second))
			assert.Equal(t, second, i.UpdatedAt())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("現在の更新日時より前の時刻を渡すとErrInvalidTimeを返し更新日時は変わらない", func(t *testing.T) {
			t.Parallel()
			at := time.Date(2026, time.September, 1, 10, 0, 0, 0, time.UTC)
			i, err := Reconstruct(
				uuidtestkit.NewTestFromSalt(t, "id"),
				Attributes{UserID: uuidtestkit.NewTestFromSalt(t, "id"), CreatedAt: at, UpdatedAt: at},
			)
			require.NoError(t, err)

			require.ErrorIs(t, i.Touch(at.Add(-time.Nanosecond)), ErrInvalidTime)
			assert.Equal(t, at, i.UpdatedAt())
		})
	})
}
