package cart

import (
	"testing"
	"time"

	"go-boilerplate/internal/domain/lexicon/money"
	"go-boilerplate/pkg/decimal"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newEvalPrice(t *testing.T, amount string) money.Price {
	t.Helper()
	d, err := decimal.Parse(amount)
	require.NoError(t, err)
	p, err := money.NewPrice(d)
	require.NoError(t, err)
	return p
}

func newEvalItem(t *testing.T, salt string, quantity int, lastSeen *money.Price) CartItem {
	t.Helper()
	return NewCartItem(uuidtestkit.NewTestFromSalt(t, salt), CartItemAttributes{
		ProductID:     uuidtestkit.NewTestFromSalt(t, salt+"_product"),
		Quantity:      quantity,
		AddedAt:       time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		LastSeenPrice: lastSeen,
	})
}

func newEvalSnapshot(t *testing.T, amount string, stock int, published bool) *ProductSnapshot {
	t.Helper()
	s := NewProductSnapshot(stock, newEvalPrice(t, amount), published)
	return &s
}

func TestNewProductSnapshot(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("観測した値をそのまま保持する", func(t *testing.T) {
			t.Parallel()

			price := newEvalPrice(t, "19.99")

			actual := NewProductSnapshot(5, price, true)

			assert.Equal(t, price, actual.Price())
		})
	})
}

func TestProductSnapshot_Price(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("観測した単価を返す", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, "19.99", NewProductSnapshot(1, newEvalPrice(t, "19.99"), true).Price().String())
		})
	})
}

func TestEvaluation_Issues(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("立った issue を返す", func(t *testing.T) {
			t.Parallel()

			actual := newEvalItem(t, "eval_issues", 1, nil).Evaluate(nil)

			assert.Equal(t, []Issue{IssueNotFound}, actual.Issues())
		})

		t.Run("問題が無ければ空を返す", func(t *testing.T) {
			t.Parallel()

			actual := newEvalItem(t, "eval_issues_ok", 1, nil).Evaluate(newEvalSnapshot(t, "10.00", 5, true))

			assert.Empty(t, actual.Issues())
		})

		t.Run("問題が無くても nil ではない", func(t *testing.T) {
			t.Parallel()

			actual := newEvalItem(t, "eval_issues_non_nil", 1, nil).Evaluate(newEvalSnapshot(t, "10.00", 5, true))

			// nil はゼロ値（未突き合わせ）の印なので、突き合わせ済みなら必ず非 nil になる。
			assert.NotNil(t, actual.Issues())
		})

		t.Run("戻り値を書き換えても内部状態は変わらない", func(t *testing.T) {
			t.Parallel()

			actual := newEvalItem(t, "eval_issues_immutable", 1, nil).Evaluate(nil)

			got := actual.Issues()
			got[0] = IssueOutOfStock

			assert.Equal(t, []Issue{IssueNotFound}, actual.Issues())
		})
	})
}

func TestEvaluation_AvailableQuantity(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("在庫不足のときは今買える上限を返す", func(t *testing.T) {
			t.Parallel()

			actual := newEvalItem(t, "eval_avail", 3, nil).Evaluate(newEvalSnapshot(t, "10.00", 1, true))

			require.NotNil(t, actual.AvailableQuantity())
			assert.Equal(t, 1, *actual.AvailableQuantity())
		})

		t.Run("在庫不足でなければ nil を返す", func(t *testing.T) {
			t.Parallel()

			actual := newEvalItem(t, "eval_avail_none", 1, nil).Evaluate(newEvalSnapshot(t, "10.00", 5, true))

			assert.Nil(t, actual.AvailableQuantity())
		})

		t.Run("戻り値を書き換えても内部状態は変わらない", func(t *testing.T) {
			t.Parallel()

			actual := newEvalItem(t, "eval_avail_immutable", 3, nil).Evaluate(newEvalSnapshot(t, "10.00", 1, true))

			got := actual.AvailableQuantity()
			require.NotNil(t, got)
			*got = 999

			require.NotNil(t, actual.AvailableQuantity())
			assert.Equal(t, 1, *actual.AvailableQuantity())
		})
	})
}

func TestEvaluation_hasNoIssue(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("issue が立っていなければ true を返す", func(t *testing.T) {
			t.Parallel()

			actual := newEvalItem(t, "eval_no_issue", 1, nil).Evaluate(newEvalSnapshot(t, "10.00", 5, true))

			assert.True(t, actual.hasNoIssue())
		})

		t.Run("issue が 1 つでも立っていれば false を返す", func(t *testing.T) {
			t.Parallel()

			actual := newEvalItem(t, "eval_has_issue", 1, nil).Evaluate(newEvalSnapshot(t, "10.00", 0, true))

			assert.False(t, actual.hasNoIssue())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("突き合わせていないゼロ値は false を返す", func(t *testing.T) {
			t.Parallel()

			assert.False(t, Evaluation{}.hasNoIssue())
		})
	})
}

func TestCartItem_Evaluate(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("在庫も価格も問題なければ issue は立たない", func(t *testing.T) {
			t.Parallel()

			seen := newEvalPrice(t, "19.99")

			actual := newEvalItem(t, "ev_ok", 2, &seen).Evaluate(newEvalSnapshot(t, "19.99", 5, true))

			assert.Empty(t, actual.Issues())
			assert.Nil(t, actual.AvailableQuantity())
		})

		t.Run("商品を引けない場合は notFound だけを立てる", func(t *testing.T) {
			t.Parallel()

			seen := newEvalPrice(t, "19.99")

			// 在庫も価格も判定材料が無いため、他の issue は立てられない。
			actual := newEvalItem(t, "ev_notfound", 2, &seen).Evaluate(nil)

			assert.Equal(t, []Issue{IssueNotFound}, actual.Issues())
			assert.Nil(t, actual.AvailableQuantity())
		})

		t.Run("非公開の商品は unpublished を立てつつ在庫と価格も判定する", func(t *testing.T) {
			t.Parallel()

			seen := newEvalPrice(t, "10.00")

			actual := newEvalItem(t, "ev_unpub", 2, &seen).Evaluate(newEvalSnapshot(t, "12.00", 0, false))

			assert.Equal(t, []Issue{IssueUnpublished, IssueOutOfStock, IssuePriceIncreased}, actual.Issues())
		})

		t.Run("在庫 0 は outOfStock のみで insufficientStock を立てない", func(t *testing.T) {
			t.Parallel()

			actual := newEvalItem(t, "ev_oos", 3, nil).Evaluate(newEvalSnapshot(t, "10.00", 0, true))

			assert.Equal(t, []Issue{IssueOutOfStock}, actual.Issues())
			assert.Nil(t, actual.AvailableQuantity())
		})

		t.Run("在庫が数量に満たない場合は insufficientStock と購入可能上限を返す", func(t *testing.T) {
			t.Parallel()

			actual := newEvalItem(t, "ev_insufficient", 3, nil).Evaluate(newEvalSnapshot(t, "10.00", 1, true))

			assert.Equal(t, []Issue{IssueInsufficientStock}, actual.Issues())
			require.NotNil(t, actual.AvailableQuantity())
			assert.Equal(t, 1, *actual.AvailableQuantity())
		})

		t.Run("在庫が数量ちょうどの場合は在庫の issue を立てない", func(t *testing.T) {
			t.Parallel()

			actual := newEvalItem(t, "ev_exact", 3, nil).Evaluate(newEvalSnapshot(t, "10.00", 3, true))

			assert.Empty(t, actual.Issues())
		})

		t.Run("提示済み価格より高ければ priceIncreased を立てる", func(t *testing.T) {
			t.Parallel()

			seen := newEvalPrice(t, "18.00")

			actual := newEvalItem(t, "ev_up", 1, &seen).Evaluate(newEvalSnapshot(t, "20.00", 5, true))

			assert.Equal(t, []Issue{IssuePriceIncreased}, actual.Issues())
		})

		t.Run("提示済み価格より安ければ priceDecreased を立てる", func(t *testing.T) {
			t.Parallel()

			seen := newEvalPrice(t, "20.00")

			actual := newEvalItem(t, "ev_down", 1, &seen).Evaluate(newEvalSnapshot(t, "18.00", 5, true))

			assert.Equal(t, []Issue{IssuePriceDecreased}, actual.Issues())
		})

		t.Run("提示済み価格と同額なら価格の issue を立てない", func(t *testing.T) {
			t.Parallel()

			seen := newEvalPrice(t, "20.00")

			actual := newEvalItem(t, "ev_same", 1, &seen).Evaluate(newEvalSnapshot(t, "20.00", 5, true))

			assert.Empty(t, actual.Issues())
		})

		t.Run("初回表示は価格差を判定しない", func(t *testing.T) {
			t.Parallel()

			// 比較の基準が無い状態で「値上がり」と言えないため、提示済み価格が無ければ判定しない。
			actual := newEvalItem(t, "ev_first", 1, nil).Evaluate(newEvalSnapshot(t, "999.00", 5, true))

			assert.Empty(t, actual.Issues())
		})
	})
}
