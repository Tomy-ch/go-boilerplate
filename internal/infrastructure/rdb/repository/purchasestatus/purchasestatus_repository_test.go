package purchasestatus

import (
	"context"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/purchase/status"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/testkit"
	"go-boilerplate/internal/observability"
	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func Test_repository_FindAll(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)

	txm := testkit.NewTestTransactionRunner(t)

	repo := &repository{
		tracer: lt,
		db:     testDB,
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("全購入ステータスが9件sortKey昇順で取得できる", func(t *testing.T) {
			t.Parallel()

			unprocessedID, err := uuid.Parse("a66c996c-86b2-41d8-9bdd-9b685fb7c47d")
			require.NoError(t, err)

			txm.WithinTx(func(ctx context.Context) {
				expectedUnprocessed, err := status.New(
					unprocessedID, status.Attributes{Name: "未処理", Code: 1, SortKey: 1},
				)
				require.NoError(t, err)

				actual, err := repo.FindAll(ctx)
				require.NoError(t, err)
				assert.Len(t, actual, 9)

				// sort_key は UNIQUE のため厳密昇順が正。
				for i := 1; i < len(actual); i++ {
					assert.Less(t, actual[i-1].SortKey(), actual[i].SortKey())
				}

				// sortKey 昇順の先頭は「未処理」(sortKey=1)。
				assert.Equal(t, expectedUnprocessed, actual[0])
			})
		})

		t.Run("code と sortKey が異なる行を、列の対応どおりに写す", func(t *testing.T) {
			t.Parallel()

			// seed は 9 件とも code と sort_key が同じ値のため、両者を取り違えても
			// 先行の一覧検証は通ってしまう。異なる値を持つ行を tx 内で足して対応を固定する
			// （tx はロールバックされる）。
			txm.WithinTx(func(ctx context.Context) {
				id, err := uuid.Parse("00000000-0000-0000-0000-0000000000fc")
				require.NoError(t, err)

				_, execErr := driver.New(ctx, testDB).Exec(ctx,
					"INSERT INTO purchase_statuses (id, name, code, sort_key) VALUES ($1,$2,$3,$4)",
					id, "テスト対応検証ステータス", 99, 50,
				)
				require.NoError(t, execErr)

				actual, err := repo.FindAll(ctx)
				require.NoError(t, err)
				require.Len(t, actual, 10)

				// sort_key=50 は seed の 1〜9 より大きいため末尾に来る。
				added := actual[len(actual)-1]
				assert.Equal(t, id, added.ID())
				assert.Equal(t, 99, added.Code())
				assert.Equal(t, 50, added.SortKey())
			})
		})

		t.Run("テーブルが空の場合、nilではない空一覧を返す", func(t *testing.T) {
			t.Parallel()

			// TRUNCATE は依存表ごと ACCESS EXCLUSIVE を取るため、直列化の外側で走る tx と
			// deadlock（40P01）しうる。require で即死させるとトランザクションマネージャーが
			// 戻り値を受け取れず、リトライ可能と宣言済みのエラーが恒久的な失敗になるため、
			// エラーは返して再試行に委ねる（WithinTxE の doc 参照）。
			txm.WithinTxE(func(ctx context.Context) error {
				// purchase_statuses を参照する依存行（purchases など）ごと空にする（tx はロールバックされる）。
				if _, execErr := driver.New(ctx, testDB).Exec(ctx, "TRUNCATE purchase_statuses CASCADE"); execErr != nil {
					return execErr
				}

				actual, err := repo.FindAll(ctx)
				require.NoError(t, err)
				assert.NotNil(t, actual)
				assert.Empty(t, actual)

				return nil
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済みコンテキストではErrCanceledへ正規化される", func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			actual, err := repo.FindAll(ctx)
			assert.Nil(t, actual)
			require.ErrorIs(t, err, apperror.ErrCanceled)
		})

		t.Run("取得行のドメイン化に失敗した場合、データ不整合としてErrInternalを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				invalidID, err := uuid.Parse("00000000-0000-0000-0000-0000000000fd")
				require.NoError(t, err)

				// sort_key=0 は有効範囲(1..32767)外のため、ドメイン化に失敗する。
				_, execErr := driver.New(ctx, testDB).Exec(ctx,
					"INSERT INTO purchase_statuses (id, name, code, sort_key) VALUES ($1,$2,$3,$4)",
					invalidID, "テスト無効ステータス", 99, 0,
				)
				require.NoError(t, execErr)

				actual, err := repo.FindAll(ctx)
				assert.Nil(t, actual)
				require.ErrorIs(t, err, apperror.ErrInternal)
				require.NotErrorIs(t, err, status.ErrInvalidSortKey)
			})
		})
	})
}

func Test_rowToPurchaseStatus(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("有効な行からエンティティを再構築する", func(t *testing.T) {
			t.Parallel()

			id, err := uuid.New()
			require.NoError(t, err)

			// code と sortKey は同型かつ同じ有効範囲のため、取り違えても検証を通過する。
			// 異なる値を渡して個別に突き合わせ、対応関係を固定する。
			entity, err := rowToPurchaseStatus(id, "支払い済み", int16(7), int16(3))
			require.NoError(t, err)
			require.NotNil(t, entity)
			assert.Equal(t, id, entity.ID())
			assert.Equal(t, "支払い済み", entity.Name())
			assert.Equal(t, 7, entity.Code())
			assert.Equal(t, 3, entity.SortKey())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("再構築時の検証失敗はErrInternalへ正規化され元の分類は露出しない", func(t *testing.T) {
			t.Parallel()

			id, err := uuid.New()
			require.NoError(t, err)

			// sort_key=0 は有効範囲(1〜32767)外のため domain 構築が失敗する。
			entity, err := rowToPurchaseStatus(id, "支払い済み", int16(7), int16(0))
			require.Error(t, err)
			assert.Nil(t, entity)
			require.ErrorIs(t, err, apperror.ErrInternal)
		})
	})
}
