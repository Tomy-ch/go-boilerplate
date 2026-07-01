package user

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
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

// insertInvalidFeedUser は、last_name が空（ドメイン不変条件違反）のユーザーを挿入するヘルパーです。
// FindFeed が行→エンティティ変換でエラーになる経路を検証するために使用します。
func insertInvalidFeedUser(ctx context.Context, t *testing.T, db driver.DBTX, id string, createdAt time.Time) {
	t.Helper()
	_, err := db.Exec(ctx,
		"INSERT INTO users "+
			"(id, first_name, last_name, password_hash, email, phone, prefecture_id, city, street, postal_code, created_at, updated_at) "+
			"VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)",
		id,
		"Feed",
		"", // last_name 空 = user.ErrInvalidLastName を誘発する。
		"$2a$08$dummydummydummydummydummydummydummydummydummydummydu",
		"feed-"+id+"@example.com",
		"000-000-0000",
		"a03aaec4-3bd6-4bfb-8e47-2fbfa026d344",
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
				drv := driver.New(ctx, testDB)
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
				drv := driver.New(ctx, testDB)
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

		t.Run("最古の行をafterカーソルにするとそれより後ろが無く空ページを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				// seed を含む全ユーザーより確実に古い行を挿入し、それをカーソル境界にする。
				// (created_at, id) < 境界 を満たす行が存在しないため、末尾到達で空ページになる。
				oldest := "ffffffff-0000-4000-8000-000000000000"
				oldestAt := time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC)
				insertFeedUser(ctx, t, driver.New(ctx, testDB), oldest, oldestAt)

				afterCursor := user.NewFeedCursor(oldestAt, mustParse(oldest))
				page, err := repo.FindFeed(ctx, &afterCursor, 10)
				require.NoError(t, err)
				assert.Empty(t, page)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("先頭ページでlimitが負数の場合、ErrInternalへ正規化される", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				// 負数 LIMIT は PostgreSQL の 2201W（map 未定義）となり ErrInternal へ写像される。
				actual, err := repo.FindFeed(ctx, nil, -1)
				require.Nil(t, actual)
				require.ErrorIs(t, err, apperror.ErrInternal)
			})
		})

		t.Run("afterカーソル指定でlimitが負数の場合、ErrInternalへ正規化される", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				afterCursor := user.NewFeedCursor(base, mustParse(tieHigh))
				after := &afterCursor
				actual, err := repo.FindFeed(ctx, after, -1)
				require.Nil(t, actual)
				require.ErrorIs(t, err, apperror.ErrInternal)
			})
		})

		t.Run("先頭ページで不正な行が含まれるとドメイン変換でErrInvalidLastNameを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				// last_name 空の行を最新時刻で挿入し、先頭ページ（after=nil）に必ず含める。
				invalidID := "ffffffff-0000-4000-8000-0000000000a1"
				insertInvalidFeedUser(ctx, t, driver.New(ctx, testDB), invalidID, base)

				actual, err := repo.FindFeed(ctx, nil, 10)
				require.Nil(t, actual)
				require.ErrorIs(t, err, user.ErrInvalidLastName)
			})
		})

		t.Run("afterカーソル指定で不正な行が含まれるとドメイン変換でErrInvalidLastNameを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				// 不正行を base に置き、それより新しい時刻をカーソル境界にして次ページへ確実に含める。
				invalidID := "ffffffff-0000-4000-8000-0000000000a2"
				insertInvalidFeedUser(ctx, t, driver.New(ctx, testDB), invalidID, base)

				afterCursor := user.NewFeedCursor(base.Add(time.Hour), mustParse(invalidID))
				after := &afterCursor
				actual, err := repo.FindFeed(ctx, after, 10)
				require.Nil(t, actual)
				require.ErrorIs(t, err, user.ErrInvalidLastName)
			})
		})
	})
}
