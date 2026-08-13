package cart

import (
	"context"
	"strings"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	domaincart "go-boilerplate/internal/domain/cart"
	"go-boilerplate/internal/domain/lexicon/money"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/testkit"
	"go-boilerplate/internal/observability"
	"go-boilerplate/pkg/decimal"
	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 既存 seed の FK 対象。
const (
	seedStatusInStock = "093170fb-83a2-4864-a2b3-53236eaf3597"
	seedCategory      = "5dd52d84-78eb-4a52-ba0b-2e11c95c2af2"
	seedUserID        = "550e8400-e29b-41d4-a716-446655440000"
)

// baseTime は、テストで用いる基準時刻です。
// carts.created_at は DB が NOW() を刻むため、集約の不変条件（有効期限は作成日時より後）を
// 満たすには基準を現在より後に置く必要があります。各ケースはこの値からの相対で時刻を組み立てます。
// マイクロ秒へ丸めるのは、保存された値と往復後の値を等価比較できるようにするためです。
var baseTime = time.Now().UTC().Truncate(time.Microsecond).Add(time.Hour)

func mustParse(t *testing.T, s string) uuid.UUID {
	t.Helper()
	id, err := uuid.Parse(s)
	require.NoError(t, err)
	return id
}

func mustNewUUID(t *testing.T) uuid.UUID {
	t.Helper()
	id, err := uuid.New()
	require.NoError(t, err)
	return id
}

// newTestToken は、指定した接尾辞で終わる規定長のセッショントークンを作ります。
func newTestToken(t *testing.T, suffix string) domaincart.SessionToken {
	t.Helper()
	value := strings.Repeat("a", 43-len(suffix)) + suffix
	token, err := domaincart.NewSessionToken(value)
	require.NoError(t, err)
	return token
}

// insertProduct は、明細の FK を満たす商品を挿入し、その ID を返します。
func insertProduct(ctx context.Context, t *testing.T, db driver.DBTX, seed string) uuid.UUID {
	t.Helper()
	productID := mustParse(t, seed)
	_, err := db.Exec(ctx,
		"INSERT INTO products (id, name, description, price, quantity, stock_warning_threshold, status_id, category_id, published_at) "+
			"VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())",
		productID, "cart-repo-test-"+seed, nil, 1000, 20, nil, seedStatusInStock, seedCategory,
	)
	require.NoError(t, err)
	return productID
}

// newGuestCart は、明細を持たないゲストカートを組み立てます（永続化はしません）。
func newGuestCart(t *testing.T, suffix string) *domaincart.Cart {
	t.Helper()
	c, err := domaincart.NewForGuest(mustNewUUID(t), newTestToken(t, suffix), baseTime.Add(24*time.Hour))
	require.NoError(t, err)
	return c
}

func TestNew(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	tf := observability.NewNoopTracerFactory(t)
	expected := &repository{
		tracer: tf.Infra(),
		db:     testDB,
	}
	actual := New(testDB, tf)
	assert.Equal(t, expected, actual)
}

func Test_repository_Create(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	repo := &repository{tracer: observability.NewMockInfraLayerTracer(t), db: testDB}
	txm := testkit.NewTestTransactionRunner(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ゲストカートを明細込みで登録する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				productID := insertProduct(ctx, t, driver.New(ctx, testDB), "c1000000-0000-4000-8000-000000000001")
				c := newGuestCart(t, "cr1")
				require.NoError(t, c.SetItem(domaincart.SetItemAttributes{ItemID: mustNewUUID(t), ProductID: productID, Quantity: 3}, baseTime))

				require.NoError(t, repo.Create(ctx, c))

				got, err := repo.FindBySessionToken(ctx, *c.SessionToken())
				require.NoError(t, err)
				assert.Equal(t, c.ID(), got.ID())
				assert.Nil(t, got.OwnerID())
				require.Len(t, got.Items(), 1)
				assert.Equal(t, productID, got.Items()[0].ProductID())
				assert.Equal(t, 3, got.Items()[0].Quantity())
			})
		})

		t.Run("所有者付きの空カートを登録する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				ownerID := mustParse(t, seedUserID)
				c, err := domaincart.NewForOwner(mustNewUUID(t), ownerID, baseTime.Add(24*time.Hour))
				require.NoError(t, err)

				require.NoError(t, repo.Create(ctx, c))

				got, ferr := repo.FindByOwnerID(ctx, ownerID)
				require.NoError(t, ferr)
				assert.Equal(t, ownerID, *got.OwnerID())
				assert.Nil(t, got.SessionToken())
				assert.True(t, got.IsEmpty())
			})
		})

		t.Run("提示価格を保存し復元できる", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				productID := insertProduct(ctx, t, driver.New(ctx, testDB), "c1000000-0000-4000-8000-000000000002")
				c := newGuestCart(t, "cr2")
				require.NoError(t, c.SetItem(domaincart.SetItemAttributes{ItemID: mustNewUUID(t), ProductID: productID, Quantity: 1}, baseTime))
				price, err := money.NewPrice(decimal.FromInt(1234))
				require.NoError(t, err)
				c.MarkSeen(map[uuid.UUID]money.Price{productID: price})

				require.NoError(t, repo.Create(ctx, c))

				got, ferr := repo.FindBySessionToken(ctx, *c.SessionToken())
				require.NoError(t, ferr)
				require.NotNil(t, got.Items()[0].LastSeenPrice())
				assert.Equal(t, "1234", got.Items()[0].LastSeenPrice().String())
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("同一セッショントークンの二重登録はConflictを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				first := newGuestCart(t, "dup")
				require.NoError(t, repo.Create(ctx, first))

				second, err := domaincart.NewForGuest(
					mustNewUUID(t), *first.SessionToken(), baseTime.Add(24*time.Hour),
				)
				require.NoError(t, err)

				require.ErrorIs(t, repo.Create(ctx, second), apperror.ErrConflict)
			})
		})

		t.Run("同一所有者の二重登録はConflictを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				ownerID := mustParse(t, seedUserID)
				first, err := domaincart.NewForOwner(mustNewUUID(t), ownerID, baseTime.Add(24*time.Hour))
				require.NoError(t, err)
				require.NoError(t, repo.Create(ctx, first))

				second, nerr := domaincart.NewForOwner(mustNewUUID(t), ownerID, baseTime.Add(24*time.Hour))
				require.NoError(t, nerr)

				require.ErrorIs(t, repo.Create(ctx, second), apperror.ErrConflict)
			})
		})

		t.Run("存在しない商品の明細はエラーを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				c := newGuestCart(t, "fk1")
				require.NoError(t, c.SetItem(domaincart.SetItemAttributes{ItemID: mustNewUUID(t), ProductID: mustNewUUID(t), Quantity: 1}, baseTime))

				require.Error(t, repo.Create(ctx, c))
			})
		})
	})
}

func Test_repository_FindBySessionToken(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	repo := &repository{tracer: observability.NewMockInfraLayerTracer(t), db: testDB}
	txm := testkit.NewTestTransactionRunner(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("明細を追加日時の昇順で返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				older := insertProduct(ctx, t, drv, "c2000000-0000-4000-8000-000000000001")
				newer := insertProduct(ctx, t, drv, "c2000000-0000-4000-8000-000000000002")
				c := newGuestCart(t, "ord")
				require.NoError(
					t,
					c.SetItem(domaincart.SetItemAttributes{ItemID: mustNewUUID(t), ProductID: newer, Quantity: 1}, baseTime.Add(time.Hour)),
				)
				require.NoError(t, c.SetItem(domaincart.SetItemAttributes{ItemID: mustNewUUID(t), ProductID: older, Quantity: 1}, baseTime))
				require.NoError(t, repo.Create(ctx, c))

				got, err := repo.FindBySessionToken(ctx, *c.SessionToken())

				require.NoError(t, err)
				require.Len(t, got.Items(), 2)
				assert.Equal(t, older, got.Items()[0].ProductID())
				assert.Equal(t, newer, got.Items()[1].ProductID())
			})
		})

		t.Run("期限切れのカートも取り除かずに返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				c, err := domaincart.NewForGuest(mustNewUUID(t), newTestToken(t, "exp"), baseTime)
				require.NoError(t, err)
				require.NoError(t, repo.Create(ctx, c))

				got, ferr := repo.FindBySessionToken(ctx, *c.SessionToken())

				require.NoError(t, ferr)
				assert.True(t, got.IsExpired(baseTime.Add(time.Hour)))
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("存在しないトークンはNotFoundを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				_, err := repo.FindBySessionToken(ctx, newTestToken(t, "nfd"))
				require.ErrorIs(t, err, apperror.ErrNotFound)
			})
		})

		t.Run("所有者が確定したカートはトークンでは引けない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				token := newTestToken(t, "asn")
				c := newGuestCart(t, "asn")
				require.NoError(t, repo.Create(ctx, c))
				require.NoError(t, c.AssignOwner(mustParse(t, seedUserID), baseTime))
				require.NoError(t, repo.Update(ctx, c))

				_, err := repo.FindBySessionToken(ctx, token)
				require.ErrorIs(t, err, apperror.ErrNotFound)
			})
		})
	})
}

func Test_repository_FindByOwnerID(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	repo := &repository{tracer: observability.NewMockInfraLayerTracer(t), db: testDB}
	txm := testkit.NewTestTransactionRunner(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("所有者のカートを明細込みで返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				productID := insertProduct(ctx, t, driver.New(ctx, testDB), "c3000000-0000-4000-8000-000000000001")
				ownerID := mustParse(t, seedUserID)
				c, err := domaincart.NewForOwner(mustNewUUID(t), ownerID, baseTime.Add(24*time.Hour))
				require.NoError(t, err)
				require.NoError(t, c.SetItem(domaincart.SetItemAttributes{ItemID: mustNewUUID(t), ProductID: productID, Quantity: 2}, baseTime))
				require.NoError(t, repo.Create(ctx, c))

				got, ferr := repo.FindByOwnerID(ctx, ownerID)

				require.NoError(t, ferr)
				assert.Equal(t, c.ID(), got.ID())
				require.Len(t, got.Items(), 1)
				assert.Equal(t, 2, got.Items()[0].Quantity())
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("存在しない所有者はNotFoundを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				_, err := repo.FindByOwnerID(ctx, mustNewUUID(t))
				require.ErrorIs(t, err, apperror.ErrNotFound)
			})
		})
	})
}

func Test_repository_LockByID(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	repo := &repository{tracer: observability.NewMockInfraLayerTracer(t), db: testDB}
	txm := testkit.NewTestTransactionRunner(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("IDからカートを明細込みで返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				productID := insertProduct(ctx, t, driver.New(ctx, testDB), "c4000000-0000-4000-8000-000000000001")
				c := newGuestCart(t, "lck")
				require.NoError(t, c.SetItem(domaincart.SetItemAttributes{ItemID: mustNewUUID(t), ProductID: productID, Quantity: 1}, baseTime))
				require.NoError(t, repo.Create(ctx, c))

				got, err := repo.LockByID(ctx, c.ID())

				require.NoError(t, err)
				assert.Equal(t, c.ID(), got.ID())
				require.Len(t, got.Items(), 1)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("存在しないIDはNotFoundを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				_, err := repo.LockByID(ctx, mustNewUUID(t))
				require.ErrorIs(t, err, apperror.ErrNotFound)
			})
		})

		t.Run("トークンの形式が壊れた行はErrInternalへ正規化する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				id := mustNewUUID(t)
				// 規定長に満たないトークンは値オブジェクトの不変条件に違反する。
				_, err := driver.New(ctx, testDB).Exec(ctx,
					"INSERT INTO carts (id, session_token, expires_at) VALUES ($1,$2,$3)",
					id, "too-short", baseTime.Add(24*time.Hour),
				)
				require.NoError(t, err)

				_, lerr := repo.LockByID(ctx, id)
				require.ErrorIs(t, lerr, apperror.ErrInternal)
			})
		})

		t.Run("有効期限が作成日時より後でない行はErrInternalへ正規化する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				id := mustNewUUID(t)
				// created_at は DB が NOW() を刻むため、過去の有効期限は集約の不変条件に違反する。
				_, err := driver.New(ctx, testDB).Exec(ctx,
					"INSERT INTO carts (id, session_token, expires_at) VALUES ($1,$2,$3)",
					id, newTestToken(t, "brk").Value(), time.Now().UTC().Add(-time.Hour),
				)
				require.NoError(t, err)

				_, lerr := repo.LockByID(ctx, id)
				require.ErrorIs(t, lerr, apperror.ErrInternal)
			})
		})

		t.Run("数量が範囲外の明細を持つ行はErrInternalへ正規化する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				productID := insertProduct(ctx, t, drv, "c4000000-0000-4000-8000-000000000002")
				c := newGuestCart(t, "qty")
				require.NoError(t, repo.Create(ctx, c))
				// 数量の上限はドメインだけが持ち、DB の CHECK は下限しか見ていない。
				_, err := drv.Exec(ctx,
					"INSERT INTO cart_items (id, cart_id, product_id, quantity) VALUES ($1,$2,$3,$4)",
					mustNewUUID(t), c.ID(), productID, 1000,
				)
				require.NoError(t, err)

				_, lerr := repo.LockByID(ctx, c.ID())
				require.ErrorIs(t, lerr, apperror.ErrInternal)
			})
		})
	})
}

func Test_repository_Update(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	repo := &repository{tracer: observability.NewMockInfraLayerTracer(t), db: testDB}
	txm := testkit.NewTestTransactionRunner(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("明細の置換で追加日時を保持する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				productID := insertProduct(ctx, t, driver.New(ctx, testDB), "c5000000-0000-4000-8000-000000000001")
				c := newGuestCart(t, "up1")
				itemID := mustNewUUID(t)
				require.NoError(t, c.SetItem(domaincart.SetItemAttributes{ItemID: itemID, ProductID: productID, Quantity: 1}, baseTime))
				require.NoError(t, repo.Create(ctx, c))

				require.NoError(
					t,
					c.SetItem(domaincart.SetItemAttributes{ItemID: mustNewUUID(t), ProductID: productID, Quantity: 9}, baseTime.Add(time.Hour)),
				)
				require.NoError(t, repo.Update(ctx, c))

				got, err := repo.FindBySessionToken(ctx, *c.SessionToken())
				require.NoError(t, err)
				require.Len(t, got.Items(), 1)
				assert.Equal(t, 9, got.Items()[0].Quantity())
				assert.Equal(t, itemID, got.Items()[0].ID())
				assert.Equal(t, baseTime.UTC(), got.Items()[0].AddedAt().UTC())
			})
		})

		t.Run("集合から消えた明細を取り除く", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				kept := insertProduct(ctx, t, drv, "c5000000-0000-4000-8000-000000000002")
				removed := insertProduct(ctx, t, drv, "c5000000-0000-4000-8000-000000000003")
				c := newGuestCart(t, "up2")
				require.NoError(t, c.SetItem(domaincart.SetItemAttributes{ItemID: mustNewUUID(t), ProductID: kept, Quantity: 1}, baseTime))
				require.NoError(t, c.SetItem(domaincart.SetItemAttributes{ItemID: mustNewUUID(t), ProductID: removed, Quantity: 1}, baseTime))
				require.NoError(t, repo.Create(ctx, c))

				require.NoError(t, c.RemoveItem(removed))
				require.NoError(t, repo.Update(ctx, c))

				got, err := repo.FindBySessionToken(ctx, *c.SessionToken())
				require.NoError(t, err)
				require.Len(t, got.Items(), 1)
				assert.Equal(t, kept, got.Items()[0].ProductID())
			})
		})

		t.Run("明細を空にすると全て取り除かれる", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				productID := insertProduct(ctx, t, driver.New(ctx, testDB), "c5000000-0000-4000-8000-000000000004")
				c := newGuestCart(t, "up3")
				require.NoError(t, c.SetItem(domaincart.SetItemAttributes{ItemID: mustNewUUID(t), ProductID: productID, Quantity: 1}, baseTime))
				require.NoError(t, repo.Create(ctx, c))

				c.Clear()
				require.NoError(t, repo.Update(ctx, c))

				got, err := repo.FindBySessionToken(ctx, *c.SessionToken())
				require.NoError(t, err)
				assert.True(t, got.IsEmpty())
			})
		})

		t.Run("所有者の確定とトークンの破棄を反映する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				ownerID := mustParse(t, seedUserID)
				c := newGuestCart(t, "up4")
				require.NoError(t, repo.Create(ctx, c))

				require.NoError(t, c.AssignOwner(ownerID, baseTime))
				require.NoError(t, repo.Update(ctx, c))

				got, err := repo.FindByOwnerID(ctx, ownerID)
				require.NoError(t, err)
				assert.Equal(t, c.ID(), got.ID())
				assert.Nil(t, got.SessionToken())
			})
		})

		t.Run("有効期限の延長を反映する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				c := newGuestCart(t, "up5")
				require.NoError(t, repo.Create(ctx, c))

				extended := baseTime.Add(72 * time.Hour)
				c.Touch(extended, 0)
				require.NoError(t, repo.Update(ctx, c))

				got, err := repo.FindBySessionToken(ctx, *c.SessionToken())
				require.NoError(t, err)
				assert.Equal(t, extended.UTC(), got.ExpiresAt().UTC())
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("存在しないカートはNotFoundを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				c := newGuestCart(t, "nf2")
				require.ErrorIs(t, repo.Update(ctx, c), apperror.ErrNotFound)
			})
		})
	})
}

func Test_repository_Delete(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	repo := &repository{tracer: observability.NewMockInfraLayerTracer(t), db: testDB}
	txm := testkit.NewTestTransactionRunner(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("カートを明細ごと削除する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				productID := insertProduct(ctx, t, drv, "c6000000-0000-4000-8000-000000000001")
				c := newGuestCart(t, "del")
				require.NoError(t, c.SetItem(domaincart.SetItemAttributes{ItemID: mustNewUUID(t), ProductID: productID, Quantity: 1}, baseTime))
				require.NoError(t, repo.Create(ctx, c))

				require.NoError(t, repo.Delete(ctx, c.ID()))

				_, err := repo.FindBySessionToken(ctx, *c.SessionToken())
				require.ErrorIs(t, err, apperror.ErrNotFound)

				var remaining int
				row := drv.QueryRow(ctx, "SELECT COUNT(*) FROM cart_items WHERE cart_id = $1", c.ID())
				require.NoError(t, row.Scan(&remaining))
				assert.Equal(t, 0, remaining)
			})
		})

		t.Run("存在しないカートでもエラーにしない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				require.NoError(t, repo.Delete(ctx, mustNewUUID(t)))
			})
		})
	})
}

func Test_repository_DeleteExpired(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	repo := &repository{tracer: observability.NewMockInfraLayerTracer(t), db: testDB}
	txm := testkit.NewTestTransactionRunner(t)

	// insertCartExpiringAt は、指定した有効期限のゲストカートを永続化します。
	insertCartExpiringAt := func(ctx context.Context, t *testing.T, suffix string, expiresAt time.Time) uuid.UUID {
		t.Helper()
		c, err := domaincart.NewForGuest(mustNewUUID(t), newTestToken(t, suffix), expiresAt)
		require.NoError(t, err)
		require.NoError(t, repo.Create(ctx, c))
		return c.ID()
	}

	// exists は、カートが残っているかを返します。
	exists := func(ctx context.Context, t *testing.T, id uuid.UUID) bool {
		t.Helper()
		var count int
		row := driver.New(ctx, testDB).QueryRow(ctx, "SELECT COUNT(*) FROM carts WHERE id = $1", id)
		require.NoError(t, row.Scan(&count))
		return count > 0
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("有効期限ちょうどのカートは削除しない", func(t *testing.T) {
			t.Parallel()

			// 境界の向きは Cart.IsExpired と一致していなければならない（同時刻は未失効）。
			txm.WithinTx(func(ctx context.Context) {
				id := insertCartExpiringAt(ctx, t, "eq1", baseTime)

				deleted, err := repo.DeleteExpired(ctx, baseTime, 10)

				require.NoError(t, err)
				assert.Equal(t, 0, deleted)
				assert.True(t, exists(ctx, t, id))
			})
		})

		t.Run("有効期限を過ぎたカートを削除する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				id := insertCartExpiringAt(ctx, t, "ex1", baseTime)

				// 保存される時刻はマイクロ秒までしか刻まれないため、それより細かい差は同時刻に丸められる。
				deleted, err := repo.DeleteExpired(ctx, baseTime.Add(time.Microsecond), 10)

				require.NoError(t, err)
				assert.Positive(t, deleted)
				assert.False(t, exists(ctx, t, id))
			})
		})

		t.Run("削除件数は上限で区切られる", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				insertCartExpiringAt(ctx, t, "lm1", baseTime.Add(time.Minute))
				insertCartExpiringAt(ctx, t, "lm2", baseTime.Add(2*time.Minute))
				insertCartExpiringAt(ctx, t, "lm3", baseTime.Add(3*time.Minute))

				deleted, err := repo.DeleteExpired(ctx, baseTime.Add(time.Hour), 2)

				require.NoError(t, err)
				assert.Equal(t, 2, deleted)
			})
		})

		t.Run("期限切れが無ければ0件を返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				insertCartExpiringAt(ctx, t, "nx1", baseTime.Add(24*time.Hour))

				deleted, err := repo.DeleteExpired(ctx, baseTime, 10)

				require.NoError(t, err)
				assert.Equal(t, 0, deleted)
			})
		})

		t.Run("削除は明細ごと行われる", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				productID := insertProduct(ctx, t, drv, "c7000000-0000-4000-8000-000000000001")
				c, err := domaincart.NewForGuest(mustNewUUID(t), newTestToken(t, "di1"), baseTime)
				require.NoError(t, err)
				require.NoError(t, c.SetItem(domaincart.SetItemAttributes{ItemID: mustNewUUID(t), ProductID: productID, Quantity: 1}, baseTime))
				require.NoError(t, repo.Create(ctx, c))

				_, derr := repo.DeleteExpired(ctx, baseTime.Add(time.Hour), 10)
				require.NoError(t, derr)

				var remaining int
				row := drv.QueryRow(ctx, "SELECT COUNT(*) FROM cart_items WHERE cart_id = $1", c.ID())
				require.NoError(t, row.Scan(&remaining))
				assert.Equal(t, 0, remaining)
			})
		})
	})
}

func Test_repository_reconstruct(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	repo := &repository{tracer: observability.NewMockInfraLayerTracer(t), db: testDB}
	txm := testkit.NewTestTransactionRunner(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("親行と明細を別クエリで読みGo側で結合する", func(t *testing.T) {
			t.Parallel()

			// 親と子を 1 本の JOIN にせず 2 クエリで読むため、明細を持つカートと持たないカートで
			// 親の属性が同じように復元されることを固定する。
			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				productID := insertProduct(ctx, t, drv, "c8000000-0000-4000-8000-000000000001")

				withItems := newGuestCart(t, "rc1")
				require.NoError(
					t,
					withItems.SetItem(domaincart.SetItemAttributes{ItemID: mustNewUUID(t), ProductID: productID, Quantity: 2}, baseTime),
				)
				require.NoError(t, repo.Create(ctx, withItems))

				empty := newGuestCart(t, "rc2")
				require.NoError(t, repo.Create(ctx, empty))

				gotWithItems, err := repo.LockByID(ctx, withItems.ID())
				require.NoError(t, err)
				gotEmpty, eerr := repo.LockByID(ctx, empty.ID())
				require.NoError(t, eerr)

				require.Len(t, gotWithItems.Items(), 1)
				assert.True(t, gotEmpty.IsEmpty())
				assert.Equal(t, gotWithItems.ExpiresAt().UTC(), gotEmpty.ExpiresAt().UTC())
				assert.Nil(t, gotWithItems.OwnerID())
				assert.NotNil(t, gotWithItems.SessionToken())
			})
		})

		t.Run("監査時刻をDBの値から復元する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				c := newGuestCart(t, "rc3")
				require.NoError(t, repo.Create(ctx, c))

				got, err := repo.LockByID(ctx, c.ID())

				require.NoError(t, err)
				assert.False(t, got.CreatedAt().IsZero())
				assert.False(t, got.UpdatedAt().IsZero())
			})
		})
	})
}

func Test_toCartItems(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("行が無ければ空を返す", func(t *testing.T) {
			t.Parallel()
			items, err := toCartItems(nil)
			require.NoError(t, err)
			assert.Empty(t, items)
		})
	})
}

func Test_toSessionToken(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("NULLならnilを返す", func(t *testing.T) {
			t.Parallel()
			token, err := toSessionToken(nil)
			require.NoError(t, err)
			assert.Nil(t, token)
		})

		t.Run("規定の形式ならトークンを返す", func(t *testing.T) {
			t.Parallel()
			value := newTestToken(t, "ok1").Value()

			token, err := toSessionToken(&value)

			require.NoError(t, err)
			require.NotNil(t, token)
			assert.Equal(t, value, token.Value())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("形式が不正ならエラーを返す", func(t *testing.T) {
			t.Parallel()
			value := "too-short"

			_, err := toSessionToken(&value)

			require.ErrorIs(t, err, domaincart.ErrInvalidSessionToken)
		})
	})
}

func Test_toPrice(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("NULLならnilを返す", func(t *testing.T) {
			t.Parallel()
			price, err := toPrice(nil)
			require.NoError(t, err)
			assert.Nil(t, price)
		})

		t.Run("値があれば価格を返す", func(t *testing.T) {
			t.Parallel()
			value := decimal.FromInt(1234)

			price, err := toPrice(&value)

			require.NoError(t, err)
			require.NotNil(t, price)
			assert.Equal(t, "1234", price.String())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("負の値はエラーを返す", func(t *testing.T) {
			t.Parallel()
			value := decimal.FromInt(-1)

			_, err := toPrice(&value)

			require.Error(t, err)
		})
	})
}

func Test_sessionTokenValue(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("トークンを持たないカートはnilを返す", func(t *testing.T) {
			t.Parallel()
			c, err := domaincart.NewForOwner(mustNewUUID(t), mustNewUUID(t), baseTime.Add(time.Hour))
			require.NoError(t, err)

			assert.Nil(t, sessionTokenValue(c))
		})

		t.Run("トークンを持つカートはその文字列を返す", func(t *testing.T) {
			t.Parallel()
			c := newGuestCart(t, "stv")

			got := sessionTokenValue(c)

			require.NotNil(t, got)
			assert.Equal(t, c.SessionToken().Value(), *got)
		})
	})
}

func Test_lastSeenPriceValue(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未提示ならnilを返す", func(t *testing.T) {
			t.Parallel()
			item := domaincart.NewCartItem(mustNewUUID(t), domaincart.CartItemAttributes{Quantity: 1})

			assert.Nil(t, lastSeenPriceValue(item))
		})

		t.Run("提示済みならその値を返す", func(t *testing.T) {
			t.Parallel()
			price, err := money.NewPrice(decimal.FromInt(500))
			require.NoError(t, err)
			item := domaincart.NewCartItem(mustNewUUID(t), domaincart.CartItemAttributes{
				Quantity: 1, LastSeenPrice: &price,
			})

			got := lastSeenPriceValue(item)

			require.NotNil(t, got)
			assert.Equal(t, "500", got.String())
		})
	})
}
