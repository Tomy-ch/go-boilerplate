package product

import (
	"strings"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/kernel/money"
	decimaltestkit "go-boilerplate/pkg/decimal/testkit"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/uuid"

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
	ref, err := NewStatusRef(uuid.NewTestFromSalt(t, salt), name)
	require.NoError(t, err)
	return ref
}

// mustCategoryRef は、テスト用に有効な商品カテゴリ参照を構築します。
func mustCategoryRef(t *testing.T, salt, name string) CategoryRef {
	t.Helper()
	ref, err := NewCategoryRef(uuid.NewTestFromSalt(t, salt), name)
	require.NoError(t, err)
	return ref
}

func validProductArgs(t *testing.T) (uuid.UUID, string, *string, money.Price, int, *int, StatusRef, CategoryRef, *time.Time, *string) {
	t.Helper()
	id := uuid.NewTestFromSalt(t, "product_id")
	status := mustStatusRef(t, "product_status_id", "在庫あり")
	category := mustCategoryRef(t, "product_category_id", "電子機器")
	publishedAt := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	imagePath := "products/earphone.png"
	return id, "ワイヤレスイヤホン", ptr.To("ノイズキャンセリング対応"), mustPrice(t, "19.99"), 100, ptr.To(10), status, category, ptr.To(publishedAt), ptr.To(imagePath)
}

// updatedProductArgs は、テスト用に更新後の商品属性一式（Update の引数順）を構築します。
func updatedProductArgs(t *testing.T) (string, *string, money.Price, int, *int, StatusRef, CategoryRef, *time.Time, *string) {
	t.Helper()
	status := mustStatusRef(t, "updated_product_status_id", "在庫切れ")
	category := mustCategoryRef(t, "updated_product_category_id", "オーディオ")
	publishedAt := time.Date(2026, time.February, 3, 4, 5, 6, 0, time.UTC)
	imagePath := "products/earphone-pro.png"
	return "ワイヤレスイヤホン Pro", ptr.To("外音取り込みに対応"), mustPrice(t, "29.95"), 50, ptr.To(5), status, category, ptr.To(publishedAt), ptr.To(imagePath)
}

// newTestProduct は、テスト用に有効な商品エンティティを生成します。バージョンは initialVersion です。
func newTestProduct(t *testing.T) *Product {
	t.Helper()
	id, name, description, price, quantity, threshold, status, category, publishedAt, imagePath := validProductArgs(t)
	p, err := New(id, name, description, price, quantity, threshold, status, category, publishedAt, imagePath)
	require.NoError(t, err)
	return p
}

func TestNew(t *testing.T) {
	t.Parallel()

	id, name, description, price, quantity, threshold, status, category, publishedAt, imagePath := validProductArgs(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("全フィールドが有効な場合、Productエンティティが生成される", func(t *testing.T) {
			t.Parallel()

			actual, err := New(id, name, description, price, quantity, threshold, status, category, publishedAt, imagePath)
			require.NoError(t, err)
			assert.Equal(t, id, actual.ID())
			assert.Equal(t, name, actual.Name())
			assert.Equal(t, description, actual.Description())
			assert.Equal(t, price, actual.Price())
			assert.Equal(t, quantity, actual.Quantity())
			assert.Equal(t, threshold, actual.StockWarningThreshold())
			assert.Equal(t, status, actual.Status())
			assert.Equal(t, category, actual.Category())
			assert.Equal(t, publishedAt, actual.PublishedAt())
			assert.Equal(t, imagePath, actual.ImagePath())
			assert.Equal(t, initialVersion, actual.Version())

			// ポインタ getter は防御的コピーを返し、返り値を書き換えてもエンティティ内部は不変。
			mutatedPublishedAt := actual.PublishedAt()
			*mutatedPublishedAt = time.Time{}
			require.NotNil(t, actual.PublishedAt())
			assert.Equal(t, *publishedAt, *actual.PublishedAt())

			mutatedImagePath := actual.ImagePath()
			*mutatedImagePath = "mutated"
			require.NotNil(t, actual.ImagePath())
			assert.Equal(t, *imagePath, *actual.ImagePath())
		})

		t.Run("description・stockWarningThreshold・publishedAt・imagePathがnilでも生成される", func(t *testing.T) {
			t.Parallel()

			actual, err := New(id, name, nil, price, quantity, nil, status, category, nil, nil)
			require.NoError(t, err)
			assert.Nil(t, actual.Description())
			assert.Nil(t, actual.StockWarningThreshold())
			assert.Nil(t, actual.PublishedAt())
			assert.Nil(t, actual.ImagePath())
		})

		t.Run("priceとquantityが0でも生成される", func(t *testing.T) {
			t.Parallel()

			actual, err := New(id, name, description, mustPrice(t, "0"), minQuantity, ptr.To(minThreshold), status, category, publishedAt, imagePath)
			require.NoError(t, err)
			assert.Equal(t, "0", actual.Price().String())
			assert.Equal(t, minQuantity, actual.Quantity())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("IDがゼロ値の場合、ErrInvalidIDを返す", func(t *testing.T) {
			t.Parallel()
			actual, err := New(uuid.UUID{}, name, description, price, quantity, threshold, status, category, publishedAt, imagePath)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidID)
		})

		t.Run("nameが空の場合、ErrInvalidNameを返す", func(t *testing.T) {
			t.Parallel()
			actual, err := New(id, "", description, price, quantity, threshold, status, category, publishedAt, imagePath)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidName)
		})

		t.Run("nameが最大長を超える場合、ErrInvalidNameを返す", func(t *testing.T) {
			t.Parallel()
			actual, err := New(
				id,
				strings.Repeat("あ", maxNameLength+1),
				description,
				price,
				quantity,
				threshold,
				status,
				category,
				publishedAt,
				imagePath,
			)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidName)
		})

		t.Run("quantityが負数の場合、ErrInvalidQuantityを返す", func(t *testing.T) {
			t.Parallel()
			actual, err := New(id, name, description, price, minQuantity-1, threshold, status, category, publishedAt, imagePath)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidQuantity)
		})

		t.Run("stockWarningThresholdが負数の場合、ErrInvalidStockWarningThresholdを返す", func(t *testing.T) {
			t.Parallel()
			actual, err := New(id, name, description, price, quantity, ptr.To(minThreshold-1), status, category, publishedAt, imagePath)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidStockWarningThreshold)
		})

		t.Run("statusがゼロ値の場合、ErrInvalidStatusIDを返す", func(t *testing.T) {
			t.Parallel()
			actual, err := New(id, name, description, price, quantity, threshold, StatusRef{}, category, publishedAt, imagePath)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidStatusID)
		})

		t.Run("categoryがゼロ値の場合、ErrInvalidCategoryIDを返す", func(t *testing.T) {
			t.Parallel()
			actual, err := New(id, name, description, price, quantity, threshold, status, CategoryRef{}, publishedAt, imagePath)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidCategoryID)
		})
	})
}

func TestReconstruct(t *testing.T) {
	t.Parallel()

	id, name, description, price, quantity, threshold, status, category, publishedAt, imagePath := validProductArgs(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("永続化済みのバージョンと全フィールドを保持したProductエンティティが再構築される", func(t *testing.T) {
			t.Parallel()

			version := initialVersion + 1
			actual, err := Reconstruct(id, name, description, price, quantity, threshold, status, category, publishedAt, imagePath, version)
			require.NoError(t, err)
			assert.Equal(t, id, actual.ID())
			assert.Equal(t, name, actual.Name())
			assert.Equal(t, description, actual.Description())
			assert.Equal(t, price, actual.Price())
			assert.Equal(t, quantity, actual.Quantity())
			assert.Equal(t, threshold, actual.StockWarningThreshold())
			assert.Equal(t, status, actual.Status())
			assert.Equal(t, category, actual.Category())
			assert.Equal(t, publishedAt, actual.PublishedAt())
			assert.Equal(t, imagePath, actual.ImagePath())
			assert.Equal(t, version, actual.Version())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("versionがinitialVersion未満の場合、ErrInvalidVersionを返す", func(t *testing.T) {
			t.Parallel()

			actual, err := Reconstruct(id, name, description, price, quantity, threshold, status, category, publishedAt, imagePath, initialVersion-1)
			require.ErrorIs(t, err, ErrInvalidVersion)
			assert.Nil(t, actual)
		})
	})
}

func Test_newProduct(t *testing.T) {
	t.Parallel()

	id, name, description, price, quantity, threshold, status, category, publishedAt, imagePath := validProductArgs(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("versionがinitialVersionと等しい場合、そのバージョンで生成される", func(t *testing.T) {
			t.Parallel()

			actual, err := newProduct(id, name, description, price, quantity, threshold, status, category, publishedAt, imagePath, initialVersion)
			require.NoError(t, err)
			assert.Equal(t, initialVersion, actual.Version())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("versionがinitialVersion未満の場合、ErrInvalidVersionを返す", func(t *testing.T) {
			t.Parallel()

			actual, err := newProduct(id, name, description, price, quantity, threshold, status, category, publishedAt, imagePath, initialVersion-1)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, ErrInvalidVersion)
		})
	})
}

func Test_validateAttributes(t *testing.T) {
	t.Parallel()

	status := mustStatusRef(t, "validate_attributes_status_id", "在庫あり")
	category := mustCategoryRef(t, "validate_attributes_category_id", "電子機器")

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("nameが最小長の場合、エラーを返さない", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, validateAttributes(strings.Repeat("あ", minNameLength), 1, ptr.To(1), status, category))
		})

		t.Run("nameが最大長の場合、エラーを返さない", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, validateAttributes(strings.Repeat("あ", maxNameLength), 1, ptr.To(1), status, category))
		})

		t.Run("quantityとstockWarningThresholdが最小値の場合、エラーを返さない", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, validateAttributes("ワイヤレスイヤホン", minQuantity, ptr.To(minThreshold), status, category))
		})

		t.Run("stockWarningThresholdがnilの場合、閾値検証をスキップしてエラーを返さない", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, validateAttributes("ワイヤレスイヤホン", minQuantity, nil, status, category))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("nameが最小長未満の場合、ErrInvalidNameを返す", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, validateAttributes("", 1, ptr.To(1), status, category), ErrInvalidName)
		})

		t.Run("nameが最大長を超える場合、ErrInvalidNameを返す", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, validateAttributes(strings.Repeat("あ", maxNameLength+1), 1, ptr.To(1), status, category), ErrInvalidName)
		})

		t.Run("quantityが最小値未満の場合、ErrInvalidQuantityを返す", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, validateAttributes("ワイヤレスイヤホン", minQuantity-1, ptr.To(1), status, category), ErrInvalidQuantity)
		})

		t.Run("stockWarningThresholdが最小値未満の場合、ErrInvalidStockWarningThresholdを返す", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(
				t,
				validateAttributes("ワイヤレスイヤホン", minQuantity, ptr.To(minThreshold-1), status, category),
				ErrInvalidStockWarningThreshold,
			)
		})

		t.Run("statusがゼロ値の場合、ErrInvalidStatusIDを返す", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, validateAttributes("ワイヤレスイヤホン", minQuantity, nil, StatusRef{}, category), ErrInvalidStatusID)
		})

		t.Run("categoryがゼロ値の場合、ErrInvalidCategoryIDを返す", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, validateAttributes("ワイヤレスイヤホン", minQuantity, nil, status, CategoryRef{}), ErrInvalidCategoryID)
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
			name, description, price, quantity, threshold, status, category, publishedAt, imagePath := updatedProductArgs(t)

			err := p.Update(name, description, price, quantity, threshold, status, category, publishedAt, imagePath)
			require.NoError(t, err)
			assert.Equal(t, name, p.Name())
			assert.Equal(t, description, p.Description())
			assert.Equal(t, price, p.Price())
			assert.Equal(t, quantity, p.Quantity())
			assert.Equal(t, threshold, p.StockWarningThreshold())
			assert.Equal(t, status, p.Status())
			assert.Equal(t, category, p.Category())
			assert.Equal(t, publishedAt, p.PublishedAt())
			assert.Equal(t, imagePath, p.ImagePath())
			assert.Equal(t, initialVersion, p.Version())
		})

		t.Run("description・stockWarningThreshold・publishedAt・imagePathをnilで更新した場合、未設定になる", func(t *testing.T) {
			t.Parallel()

			p := newTestProduct(t)
			name, _, price, quantity, _, status, category, _, _ := updatedProductArgs(t)

			err := p.Update(name, nil, price, quantity, nil, status, category, nil, nil)
			require.NoError(t, err)
			assert.Nil(t, p.Description())
			assert.Nil(t, p.StockWarningThreshold())
			assert.Nil(t, p.PublishedAt())
			assert.Nil(t, p.ImagePath())
		})

		t.Run("更新後に引数のポインタを書き換えてもエンティティ内部は変わらない", func(t *testing.T) {
			t.Parallel()

			p := newTestProduct(t)
			name, description, price, quantity, threshold, status, category, publishedAt, imagePath := updatedProductArgs(t)

			err := p.Update(name, description, price, quantity, threshold, status, category, publishedAt, imagePath)
			require.NoError(t, err)

			expectedDescription, expectedThreshold := *description, *threshold
			expectedPublishedAt, expectedImagePath := *publishedAt, *imagePath

			*description = "書き換え後の説明"
			*threshold = minThreshold
			*publishedAt = time.Time{}
			*imagePath = "mutated"

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
			name, description, price, _, threshold, status, category, publishedAt, imagePath := updatedProductArgs(t)

			err := p.Update(name, description, price, minQuantity-1, threshold, status, category, publishedAt, imagePath)
			require.ErrorIs(t, err, ErrInvalidQuantity)
			assert.Equal(t, snapshot, *p)
		})

		t.Run("nameが最大長を超える場合、ErrInvalidNameを返す", func(t *testing.T) {
			t.Parallel()

			p := newTestProduct(t)
			_, description, price, quantity, threshold, status, category, publishedAt, imagePath := updatedProductArgs(t)

			err := p.Update(strings.Repeat("あ", maxNameLength+1), description, price, quantity, threshold, status, category, publishedAt, imagePath)
			require.ErrorIs(t, err, ErrInvalidName)
		})
	})
}

func TestProduct_EnsureVersion(t *testing.T) {
	t.Parallel()

	id, name, description, price, quantity, threshold, status, category, publishedAt, imagePath := validProductArgs(t)

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

			p, err := Reconstruct(id, name, description, price, quantity, threshold, status, category, publishedAt, imagePath, initialVersion+1)
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

			id, name, description, _, quantity, threshold, status, category, publishedAt, imagePath := validProductArgs(t)
			p, err := New(id, name, description, mustPrice(t, "19.995"), quantity, threshold, status, category, publishedAt, imagePath)
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

			id, name, description, price, quantity, threshold, status, category, publishedAt, imagePath := validProductArgs(t)
			p, err := Reconstruct(id, name, description, price, quantity, threshold, status, category, publishedAt, imagePath, initialVersion+2)
			require.NoError(t, err)
			assert.Equal(t, initialVersion+2, p.Version())
		})
	})
}
