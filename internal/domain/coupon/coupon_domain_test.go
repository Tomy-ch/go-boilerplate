package coupon

import (
	"testing"
	"time"

	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testIssuedAt  = time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
	testExpiresAt = time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)
)

func validCouponArgs(t *testing.T) Attributes {
	t.Helper()

	discount, err := NewRateDiscount(newTestDecimal(t, "0.10"))
	require.NoError(t, err)
	scope, err := NewCategoryScope(newTestUUID(t))
	require.NoError(t, err)

	return Attributes{
		UserID:    newTestUUID(t),
		Discount:  discount,
		Scope:     scope,
		ExpiresAt: testExpiresAt,
		IssuedAt:  testIssuedAt,
	}
}

func newTestCoupon(t *testing.T) *Coupon {
	t.Helper()
	attrs := validCouponArgs(t)
	c, err := New(newTestUUID(t), attrs)
	require.NoError(t, err)

	return c
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("全属性が有効な場合、未使用のクーポンを生成する", func(t *testing.T) {
			t.Parallel()

			id := newTestUUID(t)
			attrs := validCouponArgs(t)

			actual, err := New(id, attrs)

			require.NoError(t, err)
			assert.Equal(t, id, actual.ID())
			assert.Equal(t, attrs.UserID, actual.UserID())
			assert.Equal(t, testExpiresAt, actual.ExpiresAt())
			assert.Equal(t, testIssuedAt, actual.IssuedAt())
			assert.False(t, actual.IsUsed())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("IDがゼロ値の場合、ErrInvalidIDを返す", func(t *testing.T) {
			t.Parallel()

			attrs := validCouponArgs(t)

			actual, err := New(uuid.UUID{}, attrs)

			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidID)
		})

		t.Run("受給者が未設定の場合、ErrInvalidUserIDを返す", func(t *testing.T) {
			t.Parallel()

			attrs := validCouponArgs(t)
			attrs.UserID = uuid.UUID{}

			actual, err := New(newTestUUID(t), attrs)

			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidUserID)
		})

		t.Run("値引きが未設定の場合、ErrInvalidDiscountを返す", func(t *testing.T) {
			t.Parallel()

			attrs := validCouponArgs(t)
			attrs.Discount = Discount{}

			actual, err := New(newTestUUID(t), attrs)

			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidDiscount)
		})

		t.Run("適用範囲が未設定の場合、ErrInvalidScopeを返す", func(t *testing.T) {
			t.Parallel()

			attrs := validCouponArgs(t)
			attrs.Scope = Scope{}

			actual, err := New(newTestUUID(t), attrs)

			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidScope)
		})

		t.Run("発行日時がゼロ値の場合、ErrInvalidIssuedAtを返す", func(t *testing.T) {
			t.Parallel()

			attrs := validCouponArgs(t)
			attrs.IssuedAt = time.Time{}

			actual, err := New(newTestUUID(t), attrs)

			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidIssuedAt)
		})

		t.Run("有効期限がゼロ値の場合、ErrInvalidExpiresAtを返す", func(t *testing.T) {
			t.Parallel()

			attrs := validCouponArgs(t)
			attrs.ExpiresAt = time.Time{}

			actual, err := New(newTestUUID(t), attrs)

			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidExpiresAt)
		})

		t.Run("有効期限が発行日時と同時刻の場合、ErrInvalidExpiresAtを返す", func(t *testing.T) {
			t.Parallel()

			attrs := validCouponArgs(t)
			attrs.ExpiresAt = attrs.IssuedAt

			actual, err := New(newTestUUID(t), attrs)

			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidExpiresAt)
		})

		t.Run("有効期限が発行日時より前の場合、ErrInvalidExpiresAtを返す", func(t *testing.T) {
			t.Parallel()

			attrs := validCouponArgs(t)
			attrs.ExpiresAt = attrs.IssuedAt.Add(-time.Hour)

			actual, err := New(newTestUUID(t), attrs)

			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidExpiresAt)
		})
	})
}

func TestReconstruct(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未使用として再構築する", func(t *testing.T) {
			t.Parallel()

			attrs := validCouponArgs(t)

			actual, err := Reconstruct(newTestUUID(t), attrs, nil)

			require.NoError(t, err)
			assert.False(t, actual.IsUsed())
		})

		t.Run("使用済みとして再構築する", func(t *testing.T) {
			t.Parallel()

			attrs := validCouponArgs(t)
			usedAt := testIssuedAt.Add(time.Hour)

			actual, err := Reconstruct(newTestUUID(t), attrs, &usedAt)

			require.NoError(t, err)
			assert.True(t, actual.IsUsed())
			require.NotNil(t, actual.UsedAt())
			assert.Equal(t, usedAt, *actual.UsedAt())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("永続化されている属性が不正な場合、生成時と同じ検証エラーを返す", func(t *testing.T) {
			t.Parallel()

			attrs := validCouponArgs(t)
			attrs.UserID = uuid.UUID{}

			actual, err := Reconstruct(newTestUUID(t), attrs, nil)

			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidUserID)
		})
	})
}

func Test_newCoupon(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("使用日時を渡した場合、防御コピーして保持する", func(t *testing.T) {
			t.Parallel()

			attrs := validCouponArgs(t)
			usedAt := testIssuedAt.Add(time.Hour)

			actual, err := newCoupon(newTestUUID(t), attrs, &usedAt)
			require.NoError(t, err)

			usedAt = time.Time{}

			require.NotNil(t, actual.UsedAt())
			assert.Equal(t, testIssuedAt.Add(time.Hour), *actual.UsedAt())
		})
	})
}

func TestCoupon_ID(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("保持しているIDを返す", func(t *testing.T) {
			t.Parallel()

			id := newTestUUID(t)
			attrs := validCouponArgs(t)
			c, err := New(id, attrs)
			require.NoError(t, err)

			assert.Equal(t, id, c.ID())
		})
	})
}

func TestCoupon_UserID(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("受給者のユーザーIDを返す", func(t *testing.T) {
			t.Parallel()

			attrs := validCouponArgs(t)
			c, err := New(newTestUUID(t), attrs)
			require.NoError(t, err)

			assert.Equal(t, attrs.UserID, c.UserID())
		})
	})
}

func TestCoupon_Discount(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("保持している値引きを返す", func(t *testing.T) {
			t.Parallel()

			attrs := validCouponArgs(t)
			c, err := New(newTestUUID(t), attrs)
			require.NoError(t, err)

			assert.Equal(t, DiscountKindRate, c.Discount().Kind())
		})
	})
}

func TestCoupon_Scope(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("保持している適用範囲を返す", func(t *testing.T) {
			t.Parallel()

			attrs := validCouponArgs(t)
			c, err := New(newTestUUID(t), attrs)
			require.NoError(t, err)

			assert.Equal(t, ScopeKindCategory, c.Scope().Kind())
		})
	})
}

func TestCoupon_ExpiresAt(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("有効期限を返す", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testExpiresAt, newTestCoupon(t).ExpiresAt())
		})
	})
}

func TestCoupon_IssuedAt(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("発行日時を返す", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testIssuedAt, newTestCoupon(t).IssuedAt())
		})
	})
}

func TestCoupon_UsedAt(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未使用の場合はnilを返す", func(t *testing.T) {
			t.Parallel()

			assert.Nil(t, newTestCoupon(t).UsedAt())
		})

		t.Run("返り値のポインタを書き換えてもエンティティ内部は変わらない", func(t *testing.T) {
			t.Parallel()

			attrs := validCouponArgs(t)
			usedAt := testIssuedAt.Add(time.Hour)
			c, err := Reconstruct(newTestUUID(t), attrs, &usedAt)
			require.NoError(t, err)

			got := c.UsedAt()
			*got = time.Time{}

			require.NotNil(t, c.UsedAt())
			assert.Equal(t, testIssuedAt.Add(time.Hour), *c.UsedAt())
		})
	})
}

func TestCoupon_IsUsed(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未使用の場合、falseを返す", func(t *testing.T) {
			t.Parallel()

			assert.False(t, newTestCoupon(t).IsUsed())
		})

		t.Run("使用済みの場合、trueを返す", func(t *testing.T) {
			t.Parallel()

			attrs := validCouponArgs(t)
			usedAt := testIssuedAt.Add(time.Hour)
			c, err := Reconstruct(newTestUUID(t), attrs, &usedAt)
			require.NoError(t, err)

			assert.True(t, c.IsUsed())
		})
	})
}

func TestCoupon_IsExpired(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("有効期限より前の時点では、falseを返す", func(t *testing.T) {
			t.Parallel()

			assert.False(t, newTestCoupon(t).IsExpired(testExpiresAt.Add(-time.Second)))
		})

		t.Run("有効期限ちょうどの時点で、trueを返す", func(t *testing.T) {
			t.Parallel()

			assert.True(t, newTestCoupon(t).IsExpired(testExpiresAt))
		})

		t.Run("有効期限を過ぎた時点では、trueを返す", func(t *testing.T) {
			t.Parallel()

			assert.True(t, newTestCoupon(t).IsExpired(testExpiresAt.Add(time.Second)))
		})
	})
}
