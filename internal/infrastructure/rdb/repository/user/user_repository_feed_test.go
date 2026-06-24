package user

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/domain/user"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/testkit"
	"go-boilerplate/internal/observability"
	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// insertFeedUser は、keyset 検証用に created_at / id を明示したユーザーを挿入するヘルパーです。
func insertFeedUser(ctx context.Context, t *testing.T, db driver.DBTX, id string, createdAt time.Time) {
	t.Helper()
	_, err := db.Exec(ctx,
		"INSERT INTO users "+
			"(id, first_name, last_name, password_hash, email, phone, prefecture_id, city, street, postal_code, created_at, updated_at) "+
			"VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)",
		id,
		"Feed",
		"User",
		"$2a$08$dummydummydummydummydummydummydummydummydummydummydu",
		"feed-"+id+"@example.com",
		"000-000-0000",
		"a03aaec4-3bd6-4bfb-8e47-2fbfa026d344", // 既存 seed の都道府県ID
		"City",
		"Street",
		"000-0000",
		createdAt,
		createdAt,
	)
	require.NoError(t, err)
}

func Test_repository_FindFeed(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	db := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)

	repo := &repository{tracer: lt, db: testDB}

	// created_at 降順で安定した順序を作るための固定 ID（タイブレーク検証のため tie ペアを含む）。
	// base は十分未来の時刻にし、seed データより必ず後ろ（より新しい）に来るようにする。
	base := time.Date(2099, time.January, 1, 0, 0, 0, 0, time.UTC)

	// tie ペア: 同一 created_at（base）で id だけが異なる。
	// ORDER BY created_at DESC, id DESC のため、id が大きい tieHigh が先に来る。
	tieHigh := "ffffffff-0000-4000-8000-000000000002"
	tieLow := "ffffffff-0000-4000-8000-000000000001"
	// それより古い 2 件。
	mid := "ffffffff-0000-4000-8000-000000000003" // base-1h
	old := "ffffffff-0000-4000-8000-000000000004" // base-2h

	mustParse := func(s string) uuid.UUID {
		id, err := uuid.Parse(s)
		require.NoError(t, err)
		return id
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("先頭ページとafterカーソルでkeysetが安定順に次ページを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, db)
				insertFeedUser(ctx, t, drv, tieHigh, base)
				insertFeedUser(ctx, t, drv, tieLow, base)
				insertFeedUser(ctx, t, drv, mid, base.Add(-time.Hour))
				insertFeedUser(ctx, t, drv, old, base.Add(-2*time.Hour))

				// 先頭ページ（after=nil, limit=2）: 最新の 2 件 = tieHigh, tieLow。
				// 同一 created_at では id DESC のため tieHigh(...0002) が tieLow(...0001) より先。
				firstPage, err := repo.FindFeed(ctx, nil, 2)
				require.NoError(t, err)
				require.Len(t, firstPage, 2)
				assert.Equal(t, mustParse(tieHigh), firstPage[0].ID())
				assert.Equal(t, mustParse(tieLow), firstPage[1].ID())

				// 次ページ: 先頭ページ末尾行(tieLow)を境界に keyset を進める。
				// (created_at, id) < (base, tieLow) のため、同一 created_at の tieHigh は除外され、
				// より古い mid → old が返る。
				last := firstPage[len(firstPage)-1]
				afterCursor := user.NewFeedCursor(last.CreatedAt(), last.ID())
				after := &afterCursor

				secondPage, err := repo.FindFeed(ctx, after, 2)
				require.NoError(t, err)
				require.Len(t, secondPage, 2)
				assert.Equal(t, mustParse(mid), secondPage[0].ID())
				assert.Equal(t, mustParse(old), secondPage[1].ID())
			})
		})

		t.Run("afterの境界がtieペアの先頭行の場合、同一created_atのもう一方が次ページ先頭に来る", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, db)
				insertFeedUser(ctx, t, drv, tieHigh, base)
				insertFeedUser(ctx, t, drv, tieLow, base)

				// 境界を tieHigh（同一 created_at の大きい id）にすると、
				// (created_at, id) < (base, tieHigh) により tieLow のみが次ページに残る
				// = created_at が同値でも id で正しくタイブレークされることの検証。
				afterCursor := user.NewFeedCursor(base, mustParse(tieHigh))
				after := &afterCursor

				page, err := repo.FindFeed(ctx, after, 10)
				require.NoError(t, err)
				require.NotEmpty(t, page)
				assert.Equal(t, mustParse(tieLow), page[0].ID())
				// 自身（tieHigh）および新しい行は含まれない。
				for _, u := range page {
					assert.NotEqual(t, mustParse(tieHigh), u.ID())
				}
			})
		})

		t.Run("limit=0の場合、空配列になる", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				actual, err := repo.FindFeed(ctx, nil, 0)
				require.NoError(t, err)
				assert.Empty(t, actual)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("先頭ページでlimitが負数の場合、エラーになる", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				actual, err := repo.FindFeed(ctx, nil, -1)
				require.Nil(t, actual)
				require.Error(t, err)
			})
		})

		t.Run("afterカーソル指定でlimitが負数の場合、エラーになる", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				afterCursor := user.NewFeedCursor(base, mustParse(tieHigh))
				after := &afterCursor
				actual, err := repo.FindFeed(ctx, after, -1)
				require.Nil(t, actual)
				require.Error(t, err)
			})
		})
	})
}
