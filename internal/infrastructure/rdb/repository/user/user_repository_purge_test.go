package user

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/testkit"
	"go-boilerplate/internal/observability"
	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedGeneralRoleID は、既存 seed の一般ロール ID です（user_roles の FK を満たすために使います）。
const seedGeneralRoleID = "a1b2c3d4-0000-4000-8000-000000000002"

// purgeCutoff は、物理削除候補の判定に用いる打ち切り時刻です。seed の論理削除済みユーザー
// （deleted_at は 2025 年）を候補に含めないよう、それより十分過去に置いています。
var purgeCutoff = time.Date(2001, time.January, 1, 0, 0, 0, 0, time.UTC)

// insertPurgeUser は、deletedAt を明示したユーザーを挿入するヘルパーです。deletedAt=nil で未削除になります。
func insertPurgeUser(ctx context.Context, t *testing.T, db driver.DBTX, id string, deletedAt *time.Time) uuid.UUID {
	t.Helper()
	_, err := db.Exec(ctx,
		"INSERT INTO users "+
			"(id, first_name, last_name, email, phone, prefecture_id, city, street, postal_code, deleted_at) "+
			"VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)",
		id,
		"Purge",
		"User",
		"purge-"+id+"@example.com",
		"0000000000",
		"a03aaec4-3bd6-4bfb-8e47-2fbfa026d344", // 既存 seed の都道府県ID
		"City",
		"Street",
		"000-0000",
		deletedAt,
	)
	require.NoError(t, err)

	parsed, err := uuid.Parse(id)
	require.NoError(t, err)
	return parsed
}

// insertPurgeUserDependents は、ユーザーに従属する認証アイデンティティとロール割り当てを挿入するヘルパーです。
func insertPurgeUserDependents(ctx context.Context, t *testing.T, db driver.DBTX, userID uuid.UUID, identityID string) {
	t.Helper()
	_, err := db.Exec(ctx,
		"INSERT INTO user_identities (id, user_id, issuer, subject) VALUES ($1,$2,$3,$4)",
		identityID, userID, "https://purge.example.com", "sub-"+identityID,
	)
	require.NoError(t, err)

	_, err = db.Exec(ctx,
		"INSERT INTO user_roles (user_id, role_id) VALUES ($1,$2)",
		userID, seedGeneralRoleID,
	)
	require.NoError(t, err)
}

// countPurgeRows は、指定ユーザーに紐づく行数を数えるヘルパーです。
func countPurgeRows(ctx context.Context, t *testing.T, db driver.DBTX, query string, userID uuid.UUID) int64 {
	t.Helper()
	var count int64
	require.NoError(t, db.QueryRow(ctx, query, userID).Scan(&count))
	return count
}

func Test_repository_FindDeletedBefore(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)
	repo := &repository{tracer: lt, db: testDB}

	// ID の昇順が keyset の前進順になるよう、末尾だけが異なる固定 ID を用意する。
	oldest := "eeeeeeee-0000-4000-8000-000000000001"
	middle := "eeeeeeee-0000-4000-8000-000000000002"
	newest := "eeeeeeee-0000-4000-8000-000000000003"
	withinRetention := "eeeeeeee-0000-4000-8000-000000000004"
	notDeleted := "eeeeeeee-0000-4000-8000-000000000005"
	atCutoff := "eeeeeeee-0000-4000-8000-000000000006"

	deletedAt := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
	afterCutoff := time.Date(2010, time.January, 1, 0, 0, 0, 0, time.UTC)

	insertFixtures := func(ctx context.Context, t *testing.T) (uuid.UUID, uuid.UUID, uuid.UUID) {
		t.Helper()
		drv := driver.New(ctx, testDB)
		oldestID := insertPurgeUser(ctx, t, drv, oldest, &deletedAt)
		middleID := insertPurgeUser(ctx, t, drv, middle, &deletedAt)
		newestID := insertPurgeUser(ctx, t, drv, newest, &deletedAt)
		insertPurgeUser(ctx, t, drv, withinRetention, &afterCutoff)
		insertPurgeUser(ctx, t, drv, notDeleted, nil)
		return oldestID, middleID, newestID
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("cutoffより古い論理削除済みユーザーだけをID昇順で返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				oldestID, middleID, newestID := insertFixtures(ctx, t)

				got, err := repo.FindDeletedBefore(ctx, purgeCutoff, nil, 100)
				require.NoError(t, err)

				assert.Equal(t, []uuid.UUID{oldestID, middleID, newestID}, got)
			})
		})

		t.Run("retention内の論理削除済みユーザーと未削除ユーザーは返さない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				insertFixtures(ctx, t)
				withinID, err := uuid.Parse(withinRetention)
				require.NoError(t, err)
				notDeletedID, err := uuid.Parse(notDeleted)
				require.NoError(t, err)

				got, err := repo.FindDeletedBefore(ctx, purgeCutoff, nil, 100)
				require.NoError(t, err)

				assert.NotContains(t, got, withinID)
				assert.NotContains(t, got, notDeletedID)
			})
		})

		t.Run("afterIDを指定するとその境界より後ろだけを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				oldestID, middleID, newestID := insertFixtures(ctx, t)

				got, err := repo.FindDeletedBefore(ctx, purgeCutoff, &oldestID, 100)
				require.NoError(t, err)

				assert.Equal(t, []uuid.UUID{middleID, newestID}, got)
			})
		})

		t.Run("limitで取得件数を打ち切る", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				oldestID, middleID, _ := insertFixtures(ctx, t)

				got, err := repo.FindDeletedBefore(ctx, purgeCutoff, nil, 2)
				require.NoError(t, err)

				assert.Equal(t, []uuid.UUID{oldestID, middleID}, got)
			})
		})

		t.Run("候補が無ければ空を返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				insertPurgeUser(ctx, t, drv, notDeleted, nil)

				got, err := repo.FindDeletedBefore(ctx, purgeCutoff, nil, 100)
				require.NoError(t, err)

				assert.Empty(t, got)
			})
		})

		t.Run("cutoffちょうどに削除されたユーザーは候補に含まない", func(t *testing.T) {
			t.Parallel()

			// 保持期間の境界は厳密（cutoff より前のみ対象）。物理削除は取り消せないため、
			// 境界の等値を含めるか否かが反転する退行をここで固定する。
			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				atCutoffID := insertPurgeUser(ctx, t, drv, atCutoff, &purgeCutoff)

				got, err := repo.FindDeletedBefore(ctx, purgeCutoff, nil, 100)
				require.NoError(t, err)

				assert.NotContains(t, got, atCutoffID)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済みコンテキストではErrCanceledへ正規化して返す", func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(t.Context())
			cancel()

			got, err := repo.FindDeletedBefore(ctx, purgeCutoff, nil, 100)
			assert.Nil(t, got)
			require.ErrorIs(t, err, apperror.ErrCanceled)
		})
	})
}

func Test_repository_PurgeByIDs(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	lt := observability.NewMockInfraLayerTracer(t)
	txm := testkit.NewTestTransactionRunner(t)
	repo := &repository{tracer: lt, db: testDB}

	const (
		countIdentities = "SELECT COUNT(*) FROM user_identities WHERE user_id = $1"
		countRoles      = "SELECT COUNT(*) FROM user_roles WHERE user_id = $1"
		countUsers      = "SELECT COUNT(*) FROM users WHERE id = $1"
	)

	target := "eeeeeeee-1111-4000-8000-000000000001"
	survivor := "eeeeeeee-1111-4000-8000-000000000002"
	alive := "eeeeeeee-1111-4000-8000-000000000003"
	deletedAt := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("対象ユーザーを従属データごと削除し削除件数を返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				targetID := insertPurgeUser(ctx, t, drv, target, &deletedAt)
				insertPurgeUserDependents(ctx, t, drv, targetID, "eeeeeeee-2222-4000-8000-000000000001")

				got, err := repo.PurgeByIDs(ctx, []uuid.UUID{targetID})
				require.NoError(t, err)

				assert.Equal(t, int64(1), got)
				assert.Equal(t, int64(0), countPurgeRows(ctx, t, drv, countIdentities, targetID))
				assert.Equal(t, int64(0), countPurgeRows(ctx, t, drv, countRoles, targetID))
				assert.Equal(t, int64(0), countPurgeRows(ctx, t, drv, countUsers, targetID))
			})
		})

		t.Run("対象外のユーザーと従属データは残る", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				targetID := insertPurgeUser(ctx, t, drv, target, &deletedAt)
				insertPurgeUserDependents(ctx, t, drv, targetID, "eeeeeeee-2222-4000-8000-000000000002")
				survivorID := insertPurgeUser(ctx, t, drv, survivor, &deletedAt)
				insertPurgeUserDependents(ctx, t, drv, survivorID, "eeeeeeee-2222-4000-8000-000000000003")

				_, err := repo.PurgeByIDs(ctx, []uuid.UUID{targetID})
				require.NoError(t, err)

				assert.Equal(t, int64(1), countPurgeRows(ctx, t, drv, countIdentities, survivorID))
				assert.Equal(t, int64(1), countPurgeRows(ctx, t, drv, countRoles, survivorID))
				assert.Equal(t, int64(1), countPurgeRows(ctx, t, drv, countUsers, survivorID))
			})
		})

		t.Run("論理削除されていないユーザーは従属データを含め削除しない", func(t *testing.T) {
			t.Parallel()

			// 物理削除は取り消せないため、候補列挙を誤って現役ユーザーの ID が届いても消えないことを
			// SQL 側の最終防壁として固定する。users にだけガードを付けると、生存したまま
			// 認証アイデンティティとロールだけを失ったアカウントが残るため、従属データも併せて検証する。
			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				aliveID := insertPurgeUser(ctx, t, drv, alive, nil)
				insertPurgeUserDependents(ctx, t, drv, aliveID, "eeeeeeee-2222-4000-8000-000000000005")

				got, err := repo.PurgeByIDs(ctx, []uuid.UUID{aliveID})
				require.NoError(t, err)

				assert.Equal(t, int64(0), got)
				assert.Equal(t, int64(1), countPurgeRows(ctx, t, drv, countIdentities, aliveID))
				assert.Equal(t, int64(1), countPurgeRows(ctx, t, drv, countRoles, aliveID))
				assert.Equal(t, int64(1), countPurgeRows(ctx, t, drv, countUsers, aliveID))
			})
		})

		t.Run("対象が空なら何も削除せず0件を返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				survivorID := insertPurgeUser(ctx, t, drv, survivor, &deletedAt)
				insertPurgeUserDependents(ctx, t, drv, survivorID, "eeeeeeee-2222-4000-8000-000000000004")

				got, err := repo.PurgeByIDs(ctx, nil)
				require.NoError(t, err)

				assert.Equal(t, int64(0), got)
				assert.Equal(t, int64(1), countPurgeRows(ctx, t, drv, countUsers, survivorID))
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("従属データが残る参照を持つユーザーは削除できずエラーを返す", func(t *testing.T) {
			t.Parallel()

			// 購入を持つユーザーは purchases からの FK が残るため、users の削除が外部キー違反になる。
			// 購入保持ユーザーの除外は usecase の責務で、Repository は DB 制約に守られていることを固定する。
			txm.WithinTx(func(ctx context.Context) {
				drv := driver.New(ctx, testDB)
				targetID := insertPurgeUser(ctx, t, drv, target, &deletedAt)
				_, err := drv.Exec(ctx,
					"INSERT INTO purchases (id, code, user_id, status_id, subtotal_amount, tax_amount, shipping_fee, total_amount) "+
						"VALUES ($1,$2,$3,$4,$5,$6,$7,$8)",
					"eeeeeeee-3333-4000-8000-000000000001",
					"purge-repo-code-1",
					targetID,
					"a66c996c-86b2-41d8-9bdd-9b685fb7c47d", // 既存 seed の未処理ステータスID
					1000, 100, 0, 1100,
				)
				require.NoError(t, err)

				_, err = repo.PurgeByIDs(ctx, []uuid.UUID{targetID})

				require.ErrorIs(t, err, apperror.ErrInvalidArgument)
			})
		})

		t.Run("キャンセル済みコンテキストではErrCanceledへ正規化して返す", func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(t.Context())
			cancel()

			targetID, err := uuid.Parse(target)
			require.NoError(t, err)

			got, err := repo.PurgeByIDs(ctx, []uuid.UUID{targetID})
			assert.Zero(t, got)
			require.ErrorIs(t, err, apperror.ErrCanceled)
		})
	})
}
