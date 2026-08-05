package product

import (
	"strings"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/lexicon/money"
	decimaltestkit "go-boilerplate/pkg/decimal/testkit"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mustPrice は、テスト用に十進文字列から非負の money.Price を構築します。
func mustPrice(t *testing.T, s string) money.Price {
	t.Helper()
	p, err := money.NewPrice(decimaltestkit.MustParse(t, s))
	require.NoError(t, err)
	return p
}

// mustStatusRef は、テスト用に有効な商品ステータス参照を構築します。
func mustStatusRef(t *testing.T, salt, name string) StatusRef {
	t.Helper()
	ref, err := NewStatusRef(uuidtestkit.NewTestFromSalt(t, salt), name)
	require.NoError(t, err)
	return ref
}

// mustCategoryRef は、テスト用に有効な商品カテゴリ参照を構築します。
func mustCategoryRef(t *testing.T, salt, name string) CategoryRef {
	t.Helper()
	ref, err := NewCategoryRef(uuidtestkit.NewTestFromSalt(t, salt), name)
	require.NoError(t, err)
	return ref
}

// validProductArgs は、テスト用に有効な商品 ID と属性一式を構築します。
// Description と ImagePath は取り違えを検出できるよう異なる値にします。
func validProductArgs(t *testing.T) (uuid.UUID, Attributes) {
	t.Helper()
	publishedAt := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	return uuidtestkit.NewTestFromSalt(t, "product_id"), Attributes{
		Name:                  "ワイヤレスイヤホン",
		Description:           ptr.To("ノイズキャンセリング対応"),
		Price:                 mustPrice(t, "19.99"),
		Quantity:              100,
		StockWarningThreshold: ptr.To(10),
		Status:                mustStatusRef(t, "product_status_id", "在庫あり"),
		Category:              mustCategoryRef(t, "product_category_id", "電子機器"),
		PublishedAt:           ptr.To(publishedAt),
		ImagePath:             ptr.To("products/earphone.png"),
	}
}

// updatedProductAttributes は、テスト用に更新後の商品属性一式を構築します。
func updatedProductAttributes(t *testing.T) Attributes {
	t.Helper()
	publishedAt := time.Date(2026, time.February, 3, 4, 5, 6, 0, time.UTC)
	return Attributes{
		Name:                  "ワイヤレスイヤホン Pro",
		Description:           ptr.To("外音取り込みに対応"),
		Price:                 mustPrice(t, "29.95"),
		Quantity:              50,
		StockWarningThreshold: ptr.To(5),
		Status:                mustStatusRef(t, "updated_product_status_id", "在庫切れ"),
		Category:              mustCategoryRef(t, "updated_product_category_id", "オーディオ"),
		PublishedAt:           ptr.To(publishedAt),
		ImagePath:             ptr.To("products/earphone-pro.png"),
	}
}

// newTestProduct は、テスト用に有効な商品エンティティを生成します。バージョンは initialVersion です。
func newTestProduct(t *testing.T) *Product {
	t.Helper()
	id, attrs := validProductArgs(t)
	p, err := New(id, attrs)
	require.NoError(t, err)
	return p
}

func TestNew(t *testing.T) {
	t.Parallel()

	id, attrs := validProductArgs(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("全フィールドが有効な場合、Productエンティティが生成される", func(t *testing.T) {
			t.Parallel()

			actual, err := New(id, attrs)
			require.NoError(t, err)
			assert.Equal(t, id, actual.ID())
			assert.Equal(t, attrs.Name, actual.Name())
			assert.Equal(t, attrs.Description, actual.Description())
			assert.Equal(t, attrs.Price, actual.Price())
			assert.Equal(t, attrs.Quantity, actual.Quantity())
			assert.Equal(t, attrs.StockWarningThreshold, actual.StockWarningThreshold())
			assert.Equal(t, attrs.Status, actual.Status())
			assert.Equal(t, attrs.Category, actual.Category())
			assert.Equal(t, attrs.PublishedAt, actual.PublishedAt())
			assert.Equal(t, attrs.ImagePath, actual.ImagePath())
			assert.Equal(t, initialVersion, actual.Version())

			// ポインタ getter は防御的コピーを返し、返り値を書き換えてもエンティティ内部は不変。
			mutatedDescription := actual.Description()
			*mutatedDescription = "mutated"
			require.NotNil(t, actual.Description())
			assert.Equal(t, *attrs.Description, *actual.Description())

			mutatedThreshold := actual.StockWarningThreshold()
			*mutatedThreshold = minThreshold
			require.NotNil(t, actual.StockWarningThreshold())
			assert.Equal(t, *attrs.StockWarningThreshold, *actual.StockWarningThreshold())

			mutatedPublishedAt := actual.PublishedAt()
			*mutatedPublishedAt = time.Time{}
			require.NotNil(t, actual.PublishedAt())
			assert.Equal(t, *attrs.PublishedAt, *actual.PublishedAt())

			mutatedImagePath := actual.ImagePath()
			*mutatedImagePath = "mutated"
			require.NotNil(t, actual.ImagePath())
			assert.Equal(t, *attrs.ImagePath, *actual.ImagePath())
		})

		t.Run("生成後に引数のポインタを書き換えてもエンティティ内部は変わらない", func(t *testing.T) {
			t.Parallel()

			// 他の並列サブテストとポインタを共有しないよう作り直す。
			localID, localAttrs := validProductArgs(t)

			actual, err := New(localID, localAttrs)
			require.NoError(t, err)

			expectedDescription, expectedThreshold := *localAttrs.Description, *localAttrs.StockWarningThreshold
			expectedPublishedAt, expectedImagePath := *localAttrs.PublishedAt, *localAttrs.ImagePath

			*localAttrs.Description = "書き換え後の説明"
			*localAttrs.StockWarningThreshold = minThreshold
			*localAttrs.PublishedAt = time.Time{}
			*localAttrs.ImagePath = "mutated"

			require.NotNil(t, actual.Description())
			assert.Equal(t, expectedDescription, *actual.Description())
			require.NotNil(t, actual.StockWarningThreshold())
			assert.Equal(t, expectedThreshold, *actual.StockWarningThreshold())
			require.NotNil(t, actual.PublishedAt())
			assert.Equal(t, expectedPublishedAt, *actual.PublishedAt())
			require.NotNil(t, actual.ImagePath())
			assert.Equal(t, expectedImagePath, *actual.ImagePath())
		})

		t.Run("description・stockWarningThreshold・publishedAt・imagePathがnilでも生成される", func(t *testing.T) {
			t.Parallel()

			cleared := attrs
			cleared.Description = nil
			cleared.StockWarningThreshold = nil
			cleared.PublishedAt = nil
			cleared.ImagePath = nil

			actual, err := New(id, cleared)
			require.NoError(t, err)
			assert.Nil(t, actual.Description())
			assert.Nil(t, actual.StockWarningThreshold())
			assert.Nil(t, actual.PublishedAt())
			assert.Nil(t, actual.ImagePath())
		})

		t.Run("priceとquantityが0でも生成される", func(t *testing.T) {
			t.Parallel()

			minimal := attrs
			minimal.Price = mustPrice(t, "0")
			minimal.Quantity = minQuantity
			minimal.StockWarningThreshold = ptr.To(minThreshold)

			actual, err := New(id, minimal)
			require.NoError(t, err)
			assert.Equal(t, "0", actual.Price().String())
			assert.Equal(t, minQuantity, actual.Quantity())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("IDがゼロ値の場合、ErrInvalidIDを返す", func(t *testing.T) {
			t.Parallel()
			actual, err := New(uuid.UUID{}, attrs)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidID)
		})

		t.Run("nameが空の場合、ErrInvalidNameを返す", func(t *testing.T) {
			t.Parallel()
			invalid := attrs
			invalid.Name = ""
			actual, err := New(id, invalid)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidName)
		})

		t.Run("nameが最大長を超える場合、ErrInvalidNameを返す", func(t *testing.T) {
			t.Parallel()
			invalid := attrs
			invalid.Name = strings.Repeat("あ", maxNameLength+1)
			actual, err := New(id, invalid)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidName)
		})

		t.Run("quantityが負数の場合、ErrInvalidQuantityを返す", func(t *testing.T) {
			t.Parallel()
			invalid := attrs
			invalid.Quantity = minQuantity - 1
			actual, err := New(id, invalid)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidQuantity)
		})

		t.Run("stockWarningThresholdが負数の場合、ErrInvalidStockWarningThresholdを返す", func(t *testing.T) {
			t.Parallel()
			invalid := attrs
			invalid.StockWarningThreshold = ptr.To(minThreshold - 1)
			actual, err := New(id, invalid)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidStockWarningThreshold)
		})

		t.Run("stockWarningThresholdが上限超過の場合、ErrInvalidStockWarningThresholdを返す", func(t *testing.T) {
			t.Parallel()
			invalid := attrs
			invalid.StockWarningThreshold = ptr.To(maxThreshold)
			*invalid.StockWarningThreshold++
			actual, err := New(id, invalid)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidStockWarningThreshold)
		})

		t.Run("statusがゼロ値の場合、ErrInvalidStatusIDを返す", func(t *testing.T) {
			t.Parallel()
			invalid := attrs
			invalid.Status = StatusRef{}
			actual, err := New(id, invalid)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidStatusID)
		})

		t.Run("categoryがゼロ値の場合、ErrInvalidCategoryIDを返す", func(t *testing.T) {
			t.Parallel()
			invalid := attrs
			invalid.Category = CategoryRef{}
			actual, err := New(id, invalid)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidCategoryID)
		})
	})
}

func TestReconstruct(t *testing.T) {
	t.Parallel()

	id, attrs := validProductArgs(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("永続化済みのバージョンと全フィールドを保持したProductエンティティが再構築される", func(t *testing.T) {
			t.Parallel()

			version := initialVersion + 1
			actual, err := Reconstruct(id, attrs, version)
			require.NoError(t, err)
			assert.Equal(t, id, actual.ID())
			assert.Equal(t, attrs.Name, actual.Name())
			assert.Equal(t, attrs.Description, actual.Description())
			assert.Equal(t, attrs.Price, actual.Price())
			assert.Equal(t, attrs.Quantity, actual.Quantity())
			assert.Equal(t, attrs.StockWarningThreshold, actual.StockWarningThreshold())
			assert.Equal(t, attrs.Status, actual.Status())
			assert.Equal(t, attrs.Category, actual.Category())
			assert.Equal(t, attrs.PublishedAt, actual.PublishedAt())
			assert.Equal(t, attrs.ImagePath, actual.ImagePath())
			assert.Equal(t, version, actual.Version())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("versionがinitialVersion未満の場合、ErrInvalidVersionを返す", func(t *testing.T) {
			t.Parallel()

			actual, err := Reconstruct(id, attrs, initialVersion-1)
			require.ErrorIs(t, err, ErrInvalidVersion)
			assert.Nil(t, actual)
		})
	})
}

func Test_newProduct(t *testing.T) {
	t.Parallel()

	id, attrs := validProductArgs(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("versionがinitialVersionと等しい場合、そのバージョンで生成される", func(t *testing.T) {
			t.Parallel()

			actual, err := newProduct(id, attrs, initialVersion)
			require.NoError(t, err)
			assert.Equal(t, initialVersion, actual.Version())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("versionがinitialVersion未満の場合、ErrInvalidVersionを返す", func(t *testing.T) {
			t.Parallel()

			actual, err := newProduct(id, attrs, initialVersion-1)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidVersion)
		})
	})
}

func Test_validateQuantity(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("最小値の場合、エラーを返さない", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, validateQuantity(minQuantity))
		})

		t.Run("最大値の場合、エラーを返さない", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, validateQuantity(maxQuantity))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("最小値未満の場合、ErrInvalidQuantityを返す", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, validateQuantity(minQuantity-1), ErrInvalidQuantity)
		})

		t.Run("最大値を超える場合、ErrInvalidQuantityを返す", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, validateQuantity(maxQuantity+1), ErrInvalidQuantity)
		})
	})
}

func Test_validateStockWarningThreshold(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("nilの場合、未設定として扱いエラーを返さない", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, validateStockWarningThreshold(nil))
		})

		t.Run("最小値の場合、エラーを返さない", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, validateStockWarningThreshold(ptr.To(minThreshold)))
		})

		t.Run("最大値の場合、エラーを返さない", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, validateStockWarningThreshold(ptr.To(maxThreshold)))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("最小値未満の場合、ErrInvalidStockWarningThresholdを返す", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, validateStockWarningThreshold(ptr.To(minThreshold-1)), ErrInvalidStockWarningThreshold)
		})

		t.Run("最大値を超える場合、ErrInvalidStockWarningThresholdを返す", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, validateStockWarningThreshold(ptr.To(maxThreshold+1)), ErrInvalidStockWarningThreshold)
		})
	})
}

func Test_validateAttributes(t *testing.T) {
	t.Parallel()

	valid := Attributes{
		Name:                  "ワイヤレスイヤホン",
		Quantity:              1,
		StockWarningThreshold: ptr.To(1),
		Status:                mustStatusRef(t, "validate_attributes_status_id", "在庫あり"),
		Category:              mustCategoryRef(t, "validate_attributes_category_id", "電子機器"),
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("nameが最小長の場合、エラーを返さない", func(t *testing.T) {
			t.Parallel()
			attrs := valid
			attrs.Name = strings.Repeat("あ", minNameLength)
			require.NoError(t, validateAttributes(attrs))
		})

		t.Run("nameが最大長の場合、エラーを返さない", func(t *testing.T) {
			t.Parallel()
			attrs := valid
			attrs.Name = strings.Repeat("あ", maxNameLength)
			require.NoError(t, validateAttributes(attrs))
		})

		t.Run("quantityとstockWarningThresholdが最小値の場合、エラーを返さない", func(t *testing.T) {
			t.Parallel()
			attrs := valid
			attrs.Quantity = minQuantity
			attrs.StockWarningThreshold = ptr.To(minThreshold)
			require.NoError(t, validateAttributes(attrs))
		})

		t.Run("stockWarningThresholdがnilの場合、閾値検証をスキップしてエラーを返さない", func(t *testing.T) {
			t.Parallel()
			attrs := valid
			attrs.Quantity = minQuantity
			attrs.StockWarningThreshold = nil
			require.NoError(t, validateAttributes(attrs))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("nameが最小長未満の場合、ErrInvalidNameを返す", func(t *testing.T) {
			t.Parallel()
			attrs := valid
			attrs.Name = ""
			require.ErrorIs(t, validateAttributes(attrs), ErrInvalidName)
		})

		t.Run("nameが最大長を超える場合、ErrInvalidNameを返す", func(t *testing.T) {
			t.Parallel()
			attrs := valid
			attrs.Name = strings.Repeat("あ", maxNameLength+1)
			require.ErrorIs(t, validateAttributes(attrs), ErrInvalidName)
		})

		t.Run("quantityが最小値未満の場合、ErrInvalidQuantityを返す", func(t *testing.T) {
			t.Parallel()
			attrs := valid
			attrs.Quantity = minQuantity - 1
			require.ErrorIs(t, validateAttributes(attrs), ErrInvalidQuantity)
		})

		t.Run("stockWarningThresholdが最小値未満の場合、ErrInvalidStockWarningThresholdを返す", func(t *testing.T) {
			t.Parallel()
			attrs := valid
			attrs.Quantity = minQuantity
			attrs.StockWarningThreshold = ptr.To(minThreshold - 1)
			require.ErrorIs(t, validateAttributes(attrs), ErrInvalidStockWarningThreshold)
		})

		t.Run("stockWarningThresholdが最大値を超える場合、ErrInvalidStockWarningThresholdを返す", func(t *testing.T) {
			t.Parallel()
			attrs := valid
			attrs.StockWarningThreshold = ptr.To(maxThreshold + 1)
			require.ErrorIs(t, validateAttributes(attrs), ErrInvalidStockWarningThreshold)
		})

		t.Run("statusがゼロ値の場合、ErrInvalidStatusIDを返す", func(t *testing.T) {
			t.Parallel()
			attrs := valid
			attrs.Status = StatusRef{}
			require.ErrorIs(t, validateAttributes(attrs), ErrInvalidStatusID)
		})

		t.Run("categoryがゼロ値の場合、ErrInvalidCategoryIDを返す", func(t *testing.T) {
			t.Parallel()
			attrs := valid
			attrs.Category = CategoryRef{}
			require.ErrorIs(t, validateAttributes(attrs), ErrInvalidCategoryID)
		})
	})
}

func TestProduct_Update(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("全属性が有効な場合、属性が更新されバージョンは進まない", func(t *testing.T) {
			t.Parallel()

			p := newTestProduct(t)
			attrs := updatedProductAttributes(t)

			require.NoError(t, p.Update(attrs))
			assert.Equal(t, attrs.Name, p.Name())
			assert.Equal(t, attrs.Description, p.Description())
			assert.Equal(t, attrs.Price, p.Price())
			assert.Equal(t, attrs.Quantity, p.Quantity())
			assert.Equal(t, attrs.StockWarningThreshold, p.StockWarningThreshold())
			assert.Equal(t, attrs.Status, p.Status())
			assert.Equal(t, attrs.Category, p.Category())
			assert.Equal(t, attrs.PublishedAt, p.PublishedAt())
			assert.Equal(t, attrs.ImagePath, p.ImagePath())
			assert.Equal(t, initialVersion, p.Version())
		})

		t.Run("description・stockWarningThreshold・publishedAt・imagePathをnilで更新した場合、未設定になる", func(t *testing.T) {
			t.Parallel()

			p := newTestProduct(t)
			attrs := updatedProductAttributes(t)
			attrs.Description = nil
			attrs.StockWarningThreshold = nil
			attrs.PublishedAt = nil
			attrs.ImagePath = nil

			require.NoError(t, p.Update(attrs))
			assert.Nil(t, p.Description())
			assert.Nil(t, p.StockWarningThreshold())
			assert.Nil(t, p.PublishedAt())
			assert.Nil(t, p.ImagePath())
		})

		t.Run("更新後に引数のポインタを書き換えてもエンティティ内部は変わらない", func(t *testing.T) {
			t.Parallel()

			p := newTestProduct(t)
			attrs := updatedProductAttributes(t)

			require.NoError(t, p.Update(attrs))

			expectedDescription, expectedThreshold := *attrs.Description, *attrs.StockWarningThreshold
			expectedPublishedAt, expectedImagePath := *attrs.PublishedAt, *attrs.ImagePath

			*attrs.Description = "書き換え後の説明"
			*attrs.StockWarningThreshold = minThreshold
			*attrs.PublishedAt = time.Time{}
			*attrs.ImagePath = "mutated"

			require.NotNil(t, p.Description())
			assert.Equal(t, expectedDescription, *p.Description())
			require.NotNil(t, p.StockWarningThreshold())
			assert.Equal(t, expectedThreshold, *p.StockWarningThreshold())
			require.NotNil(t, p.PublishedAt())
			assert.Equal(t, expectedPublishedAt, *p.PublishedAt())
			require.NotNil(t, p.ImagePath())
			assert.Equal(t, expectedImagePath, *p.ImagePath())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("quantityが最小値未満の場合、ErrInvalidQuantityを返しエンティティを一切変更しない", func(t *testing.T) {
			t.Parallel()

			p := newTestProduct(t)
			snapshot := *p
			attrs := updatedProductAttributes(t)
			attrs.Quantity = minQuantity - 1

			require.ErrorIs(t, p.Update(attrs), ErrInvalidQuantity)
			assert.Equal(t, snapshot, *p)
		})

		t.Run("nameが最大長を超える場合、ErrInvalidNameを返す", func(t *testing.T) {
			t.Parallel()

			p := newTestProduct(t)
			attrs := updatedProductAttributes(t)
			attrs.Name = strings.Repeat("あ", maxNameLength+1)

			require.ErrorIs(t, p.Update(attrs), ErrInvalidName)
		})
	})
}

func TestProduct_AdjustStock(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("正のdeltaを渡した場合、在庫が加算されバージョンは進まない", func(t *testing.T) {
			t.Parallel()

			p := newTestProduct(t)
			before := p.Quantity()

			require.NoError(t, p.AdjustStock(50))
			assert.Equal(t, before+50, p.Quantity())
			assert.Equal(t, initialVersion, p.Version())
		})

		t.Run("負のdeltaを渡した場合、在庫が減算される", func(t *testing.T) {
			t.Parallel()

			p := newTestProduct(t)
			before := p.Quantity()

			require.NoError(t, p.AdjustStock(-30))
			assert.Equal(t, before-30, p.Quantity())
		})

		t.Run("deltaが0の場合、在庫は変わらない", func(t *testing.T) {
			t.Parallel()

			p := newTestProduct(t)
			before := p.Quantity()

			require.NoError(t, p.AdjustStock(0))
			assert.Equal(t, before, p.Quantity())
		})

		t.Run("減算後がちょうど最小値になる場合、在庫が最小値になる", func(t *testing.T) {
			t.Parallel()

			p := newTestProduct(t)

			require.NoError(t, p.AdjustStock(minQuantity-p.Quantity()))
			assert.Equal(t, minQuantity, p.Quantity())
		})

		t.Run("加算後がちょうど最大値になる場合、在庫が最大値になる", func(t *testing.T) {
			t.Parallel()

			p := newTestProduct(t)

			require.NoError(t, p.AdjustStock(maxQuantity-p.Quantity()))
			assert.Equal(t, maxQuantity, p.Quantity())
		})

		t.Run("在庫以外の属性は変わらない", func(t *testing.T) {
			t.Parallel()

			p := newTestProduct(t)
			snapshot := *p

			require.NoError(t, p.AdjustStock(1))

			snapshot.quantity = p.Quantity()
			assert.Equal(t, snapshot, *p)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("減算後が最小値未満になる場合、ErrInvalidQuantityを返しエンティティを一切変更しない", func(t *testing.T) {
			t.Parallel()

			p := newTestProduct(t)
			snapshot := *p

			err := p.AdjustStock(minQuantity - p.Quantity() - 1)
			require.ErrorIs(t, err, ErrInvalidQuantity)
			require.ErrorIs(t, err, apperror.ErrValidation)
			assert.Equal(t, snapshot, *p)
		})

		t.Run("加算後が最大値を超える場合、ErrInvalidQuantityを返しエンティティを一切変更しない", func(t *testing.T) {
			t.Parallel()

			p := newTestProduct(t)
			snapshot := *p

			err := p.AdjustStock(maxQuantity - p.Quantity() + 1)
			require.ErrorIs(t, err, ErrInvalidQuantity)
			require.ErrorIs(t, err, apperror.ErrValidation)
			assert.Equal(t, snapshot, *p)
		})
	})
}

func TestProduct_EnsureVersion(t *testing.T) {
	t.Parallel()

	id, attrs := validProductArgs(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("期待バージョンが現在のバージョンと一致する場合、nilを返す", func(t *testing.T) {
			t.Parallel()

			p := newTestProduct(t)
			require.NoError(t, p.EnsureVersion(initialVersion))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("期待バージョンが現在のバージョンと一致しない場合、Conflictに分類されるErrVersionConflictを返す", func(t *testing.T) {
			t.Parallel()

			p, err := Reconstruct(id, attrs, initialVersion+1)
			require.NoError(t, err)

			err = p.EnsureVersion(initialVersion)
			require.ErrorIs(t, err, ErrVersionConflict)
			require.ErrorIs(t, err, apperror.ErrConflict)
		})
	})
}

func TestProduct_Price(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("サブセント精度を保持した価格スナップショットを返す", func(t *testing.T) {
			t.Parallel()

			id, attrs := validProductArgs(t)
			attrs.Price = mustPrice(t, "19.995")

			p, err := New(id, attrs)
			require.NoError(t, err)
			assert.Equal(t, "19.995", p.Price().String())
			assert.True(t, p.Price().Decimal().Equal(decimaltestkit.MustParse(t, "19.995")))
		})
	})
}

func TestProduct_Version(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("再構築で渡した楽観ロックのバージョンを返す", func(t *testing.T) {
			t.Parallel()

			id, attrs := validProductArgs(t)
			p, err := Reconstruct(id, attrs, initialVersion+2)
			require.NoError(t, err)
			assert.Equal(t, initialVersion+2, p.Version())
		})
	})
}

func TestProduct_Category(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("生成時のカテゴリ参照をIDと名称ごと返す", func(t *testing.T) {
			t.Parallel()

			id, attrs := validProductArgs(t)
			p, err := New(id, attrs)
			require.NoError(t, err)

			assert.Equal(t, attrs.Category, p.Category())
			assert.Equal(t, "電子機器", p.Category().Name())
		})
	})
}

func TestProduct_Description(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("生成時の商品説明を返す", func(t *testing.T) {
			t.Parallel()

			id, attrs := validProductArgs(t)
			p, err := New(id, attrs)
			require.NoError(t, err)

			require.NotNil(t, p.Description())
			assert.Equal(t, "ノイズキャンセリング対応", *p.Description())
		})

		t.Run("未設定の場合、nilを返す", func(t *testing.T) {
			t.Parallel()

			id, attrs := validProductArgs(t)
			attrs.Description = nil
			p, err := New(id, attrs)
			require.NoError(t, err)

			assert.Nil(t, p.Description())
		})

		t.Run("返り値のポインタを書き換えてもエンティティ内部は変わらない", func(t *testing.T) {
			t.Parallel()

			id, attrs := validProductArgs(t)
			p, err := New(id, attrs)
			require.NoError(t, err)

			got := p.Description()
			*got = "書き換え後の説明"

			require.NotNil(t, p.Description())
			assert.NotEqual(t, *got, *p.Description())
			assert.Equal(t, "ノイズキャンセリング対応", *p.Description())
		})
	})
}

func TestProduct_ID(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("生成時のIDを返す", func(t *testing.T) {
			t.Parallel()

			id, attrs := validProductArgs(t)
			p, err := New(id, attrs)
			require.NoError(t, err)

			assert.Equal(t, id, p.ID())
		})
	})
}

func TestProduct_ImagePath(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("生成時の画像パスを返す", func(t *testing.T) {
			t.Parallel()

			id, attrs := validProductArgs(t)
			p, err := New(id, attrs)
			require.NoError(t, err)

			require.NotNil(t, p.ImagePath())
			assert.Equal(t, "products/earphone.png", *p.ImagePath())
		})

		t.Run("未設定の場合、nilを返す", func(t *testing.T) {
			t.Parallel()

			id, attrs := validProductArgs(t)
			attrs.ImagePath = nil
			p, err := New(id, attrs)
			require.NoError(t, err)

			assert.Nil(t, p.ImagePath())
		})

		t.Run("返り値のポインタを書き換えてもエンティティ内部は変わらない", func(t *testing.T) {
			t.Parallel()

			id, attrs := validProductArgs(t)
			p, err := New(id, attrs)
			require.NoError(t, err)

			got := p.ImagePath()
			*got = "products/mutated.png"

			require.NotNil(t, p.ImagePath())
			assert.NotEqual(t, *got, *p.ImagePath())
			assert.Equal(t, "products/earphone.png", *p.ImagePath())
		})
	})
}

func TestProduct_Name(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("生成時の商品名を返す", func(t *testing.T) {
			t.Parallel()

			id, attrs := validProductArgs(t)
			p, err := New(id, attrs)
			require.NoError(t, err)

			assert.Equal(t, "ワイヤレスイヤホン", p.Name())
		})
	})
}

func TestProduct_PublishedAt(t *testing.T) {
	t.Parallel()

	publishedAt := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("生成時の公開日時を返す", func(t *testing.T) {
			t.Parallel()

			id, attrs := validProductArgs(t)
			p, err := New(id, attrs)
			require.NoError(t, err)

			require.NotNil(t, p.PublishedAt())
			assert.Equal(t, publishedAt, *p.PublishedAt())
		})

		t.Run("未公開の場合、nilを返す", func(t *testing.T) {
			t.Parallel()

			id, attrs := validProductArgs(t)
			attrs.PublishedAt = nil
			p, err := New(id, attrs)
			require.NoError(t, err)

			assert.Nil(t, p.PublishedAt())
		})

		t.Run("返り値のポインタを書き換えてもエンティティ内部は変わらない", func(t *testing.T) {
			t.Parallel()

			id, attrs := validProductArgs(t)
			p, err := New(id, attrs)
			require.NoError(t, err)

			got := p.PublishedAt()
			*got = time.Date(2030, time.December, 31, 0, 0, 0, 0, time.UTC)

			require.NotNil(t, p.PublishedAt())
			assert.NotEqual(t, *got, *p.PublishedAt())
			assert.Equal(t, publishedAt, *p.PublishedAt())
		})
	})
}

func TestProduct_Quantity(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("生成時の在庫数を返す", func(t *testing.T) {
			t.Parallel()

			id, attrs := validProductArgs(t)
			p, err := New(id, attrs)
			require.NoError(t, err)

			assert.Equal(t, 100, p.Quantity())
		})

		t.Run("在庫数が下限値の場合、その値を返す", func(t *testing.T) {
			t.Parallel()

			id, attrs := validProductArgs(t)
			attrs.Quantity = minQuantity
			p, err := New(id, attrs)
			require.NoError(t, err)

			assert.Equal(t, minQuantity, p.Quantity())
		})
	})
}

func TestProduct_Status(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("生成時のステータス参照をIDと名称ごと返す", func(t *testing.T) {
			t.Parallel()

			id, attrs := validProductArgs(t)
			p, err := New(id, attrs)
			require.NoError(t, err)

			assert.Equal(t, attrs.Status, p.Status())
			assert.Equal(t, "在庫あり", p.Status().Name())
		})
	})
}

func TestProduct_StockWarningThreshold(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("生成時の在庫警告閾値を返す", func(t *testing.T) {
			t.Parallel()

			id, attrs := validProductArgs(t)
			p, err := New(id, attrs)
			require.NoError(t, err)

			require.NotNil(t, p.StockWarningThreshold())
			assert.Equal(t, 10, *p.StockWarningThreshold())
		})

		t.Run("未設定の場合、nilを返す", func(t *testing.T) {
			t.Parallel()

			id, attrs := validProductArgs(t)
			attrs.StockWarningThreshold = nil
			p, err := New(id, attrs)
			require.NoError(t, err)

			assert.Nil(t, p.StockWarningThreshold())
		})

		t.Run("返り値のポインタを書き換えてもエンティティ内部は変わらない", func(t *testing.T) {
			t.Parallel()

			id, attrs := validProductArgs(t)
			p, err := New(id, attrs)
			require.NoError(t, err)

			got := p.StockWarningThreshold()
			*got = minThreshold

			require.NotNil(t, p.StockWarningThreshold())
			assert.NotEqual(t, *got, *p.StockWarningThreshold())
			assert.Equal(t, 10, *p.StockWarningThreshold())
		})
	})
}

func TestProduct_IsPublished(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("公開日時が設定されている場合、trueを返す", func(t *testing.T) {
			t.Parallel()

			id, attrs := validProductArgs(t)
			p, err := New(id, attrs)
			require.NoError(t, err)

			assert.True(t, p.IsPublished())
		})

		t.Run("公開日時が未設定の場合、falseを返す", func(t *testing.T) {
			t.Parallel()

			id, attrs := validProductArgs(t)
			attrs.PublishedAt = nil
			p, err := New(id, attrs)
			require.NoError(t, err)

			assert.False(t, p.IsPublished())
		})
	})
}

func TestProduct_IsLowStock(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("在庫が閾値を下回る場合、trueを返す", func(t *testing.T) {
			t.Parallel()

			id, attrs := validProductArgs(t)
			attrs.StockWarningThreshold = ptr.To(10)
			attrs.Quantity = 9
			p, err := New(id, attrs)
			require.NoError(t, err)

			assert.True(t, p.IsLowStock())
		})

		t.Run("在庫が閾値と等しい場合、境界を含むためtrueを返す", func(t *testing.T) {
			t.Parallel()

			id, attrs := validProductArgs(t)
			attrs.StockWarningThreshold = ptr.To(10)
			attrs.Quantity = 10
			p, err := New(id, attrs)
			require.NoError(t, err)

			assert.True(t, p.IsLowStock())
		})

		t.Run("在庫が閾値を上回る場合、falseを返す", func(t *testing.T) {
			t.Parallel()

			id, attrs := validProductArgs(t)
			attrs.StockWarningThreshold = ptr.To(10)
			attrs.Quantity = 11
			p, err := New(id, attrs)
			require.NoError(t, err)

			assert.False(t, p.IsLowStock())
		})

		t.Run("閾値が未設定の場合、在庫が0でも警告対象を持たないためfalseを返す", func(t *testing.T) {
			t.Parallel()

			id, attrs := validProductArgs(t)
			attrs.StockWarningThreshold = nil
			attrs.Quantity = 0
			p, err := New(id, attrs)
			require.NoError(t, err)

			assert.False(t, p.IsLowStock())
		})
	})
}
