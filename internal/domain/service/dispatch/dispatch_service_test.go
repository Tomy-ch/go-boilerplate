package dispatch

import (
	"testing"
	"time"

	"go-boilerplate/internal/domain/lexicon/money"
	"go-boilerplate/internal/domain/purchase"
	decimaltestkit "go-boilerplate/pkg/decimal/testkit"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// baseTime は、テスト用の購入の注文日時の基準です。
var baseTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

// newShippablePurchase は、指定の購入者・注文日時を持つ発送可能な購入を再構築します。
// idSalt は購入 ID を決めるため、同時刻の並び順を検証するテストはここで順序を指定します。
func newShippablePurchase(t *testing.T, idSalt string, userID uuid.UUID, orderedAt time.Time) *purchase.Purchase {
	t.Helper()

	unitPrice, err := money.NewPrice(decimaltestkit.MustParse(t, "800"))
	require.NoError(t, err)

	paidAt := orderedAt.Add(time.Minute)
	p, err := purchase.Reconstruct(uuidtestkit.NewTestFromSalt(t, idSalt), purchase.Attributes{
		Code:       idSalt,
		UserID:     userID,
		StatusID:   uuidtestkit.NewTestFromSalt(t, "dispatch_status"),
		StatusCode: purchase.StatusPaid.Code(),
		Details: []purchase.PurchaseDetail{
			purchase.NewPurchaseDetail(
				uuidtestkit.NewTestFromSalt(t, idSalt+"_detail"),
				purchase.PurchaseDetailAttributes{
					ProductID: uuidtestkit.NewTestFromSalt(t, "dispatch_product"),
					Quantity:  1,
					UnitPrice: unitPrice,
				},
			),
		},
		SubtotalAmount: 80000,
		TaxAmount:      8000,
		ShippingFee:    500,
		TotalAmount:    88500,
		OrderedAt:      orderedAt,
		PaidAt:         &paidAt,
	})
	require.NoError(t, err)
	return p
}

// codesOf は、組の並びを購入コードの二重スライスへ落とします。順序の検証を読みやすくするためです。
func codesOf(groups []purchase.Purchases) [][]string {
	codes := make([][]string, 0, len(groups))
	for _, group := range groups {
		groupCodes := make([]string, 0, len(group))
		for _, p := range group {
			groupCodes = append(groupCodes, p.Code())
		}
		codes = append(codes, groupCodes)
	}
	return codes
}

func TestGroupForDispatch(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("購入者ごとに組が分かれる", func(t *testing.T) {
			t.Parallel()

			alice := uuidtestkit.NewTestFromSalt(t, "alice")
			bob := uuidtestkit.NewTestFromSalt(t, "bob")

			groups := GroupForDispatch(purchase.Purchases{
				newShippablePurchase(t, "a1", alice, baseTime),
				newShippablePurchase(t, "b1", bob, baseTime.Add(time.Hour)),
				newShippablePurchase(t, "a2", alice, baseTime.Add(2*time.Hour)),
			})

			assert.Equal(t, [][]string{{"a1", "a2"}, {"b1"}}, codesOf(groups))
		})

		t.Run("購入が1件だけの購入者も1件からなる組になる", func(t *testing.T) {
			t.Parallel()

			groups := GroupForDispatch(purchase.Purchases{
				newShippablePurchase(t, "solo", uuidtestkit.NewTestFromSalt(t, "solo_user"), baseTime),
			})

			assert.Equal(t, [][]string{{"solo"}}, codesOf(groups))
		})

		t.Run("組の中は注文日時の古い順に並ぶ", func(t *testing.T) {
			t.Parallel()

			alice := uuidtestkit.NewTestFromSalt(t, "alice")

			groups := GroupForDispatch(purchase.Purchases{
				newShippablePurchase(t, "late", alice, baseTime.Add(2*time.Hour)),
				newShippablePurchase(t, "early", alice, baseTime),
				newShippablePurchase(t, "middle", alice, baseTime.Add(time.Hour)),
			})

			assert.Equal(t, [][]string{{"early", "middle", "late"}}, codesOf(groups))
		})

		t.Run("組同士はその組の最も古い購入の注文日時の順に並ぶ", func(t *testing.T) {
			t.Parallel()

			alice := uuidtestkit.NewTestFromSalt(t, "alice")
			bob := uuidtestkit.NewTestFromSalt(t, "bob")

			// bob の最古(1h)は alice の最古(0h)より後だが、alice の最新(3h)よりは先。
			// 組の順序が最古で決まることを、最新で決めた場合と結果が食い違う配置で確かめる。
			groups := GroupForDispatch(purchase.Purchases{
				newShippablePurchase(t, "b1", bob, baseTime.Add(time.Hour)),
				newShippablePurchase(t, "b2", bob, baseTime.Add(2*time.Hour)),
				newShippablePurchase(t, "a1", alice, baseTime),
				newShippablePurchase(t, "a2", alice, baseTime.Add(3*time.Hour)),
			})

			assert.Equal(t, [][]string{{"a1", "a2"}, {"b1", "b2"}}, codesOf(groups))
		})

		t.Run("注文日時が同時刻の場合、購入IDの昇順に並ぶ", func(t *testing.T) {
			t.Parallel()

			alice := uuidtestkit.NewTestFromSalt(t, "alice")
			first := newShippablePurchase(t, "same_a", alice, baseTime)
			second := newShippablePurchase(t, "same_b", alice, baseTime)
			if first.ID().String() > second.ID().String() {
				first, second = second, first
			}

			groups := GroupForDispatch(purchase.Purchases{second, first})

			require.Len(t, groups, 1)
			assert.Equal(t, []string{first.Code(), second.Code()}, codesOf(groups)[0])
		})

		t.Run("入力の並び順が変わっても同じ結果を返す", func(t *testing.T) {
			t.Parallel()

			alice := uuidtestkit.NewTestFromSalt(t, "alice")
			bob := uuidtestkit.NewTestFromSalt(t, "bob")
			a1 := newShippablePurchase(t, "a1", alice, baseTime)
			a2 := newShippablePurchase(t, "a2", alice, baseTime.Add(2*time.Hour))
			b1 := newShippablePurchase(t, "b1", bob, baseTime.Add(time.Hour))

			assert.Equal(t,
				codesOf(GroupForDispatch(purchase.Purchases{a1, b1, a2})),
				codesOf(GroupForDispatch(purchase.Purchases{a2, a1, b1})),
			)
		})

		t.Run("購入が空の場合、空を返す", func(t *testing.T) {
			t.Parallel()

			assert.Empty(t, GroupForDispatch(purchase.Purchases{}))
		})
	})
}

func Test_compareDispatchOrder(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		alice := uuidtestkit.NewTestFromSalt(t, "cmp_alice")

		t.Run("注文日時が早い購入を先とする", func(t *testing.T) {
			t.Parallel()

			early := newShippablePurchase(t, "cmp_early", alice, baseTime)
			late := newShippablePurchase(t, "cmp_late", alice, baseTime.Add(time.Hour))

			assert.Negative(t, compareDispatchOrder(early, late))
			assert.Positive(t, compareDispatchOrder(late, early))
		})

		t.Run("注文日時が同時刻の場合、購入IDの小さい方を先とする", func(t *testing.T) {
			t.Parallel()

			first := newShippablePurchase(t, "cmp_same_a", alice, baseTime)
			second := newShippablePurchase(t, "cmp_same_b", alice, baseTime)
			if first.ID().String() > second.ID().String() {
				first, second = second, first
			}

			assert.Negative(t, compareDispatchOrder(first, second))
			assert.Positive(t, compareDispatchOrder(second, first))
		})

		t.Run("同一の購入同士は0を返す", func(t *testing.T) {
			t.Parallel()

			p := newShippablePurchase(t, "cmp_self", alice, baseTime)

			assert.Zero(t, compareDispatchOrder(p, p))
		})
	})
}
