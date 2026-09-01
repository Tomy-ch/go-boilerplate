package inquiry

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/apperror"
	domaininquiry "go-boilerplate/internal/domain/inquiry"
	"go-boilerplate/internal/infrastructure/rdb/testkit"
	"go-boilerplate/internal/observability"
	"go-boilerplate/pkg/uuid"
)

// 既存 seed のユーザー。inquiries.user_id は UNIQUE のため、並行するケースは別々の利用者を使います。
const (
	seedUserA = "550e8400-e29b-41d4-a716-446655440000"
	seedUserB = "faba7bb2-f5a0-4a51-adae-1564929077b2"
	seedUserC = "a95a2dd3-2b37-4def-8041-23d2138faccc"
	seedUserD = "705349ae-d6d8-48ad-8263-6ab48cc9201b"
	seedUserE = "0b393ac1-b8a2-4f69-8972-de680aeb0a95"
	seedUserF = "212f525e-ffab-4523-9ec0-8af76e006fe3"
	seedUserG = "090f5b51-37ac-4413-b326-1709ae4661f4"
)

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

// createInquiry は、指定した利用者の問い合わせを 1 件登録し、その集約を返します。
func createInquiry(ctx context.Context, t *testing.T, repo *repository, seedUser string) *domaininquiry.Inquiry {
	t.Helper()
	i, err := domaininquiry.New(mustNewUUID(t), domaininquiry.Attributes{UserID: mustParse(t, seedUser)})
	require.NoError(t, err)
	require.NoError(t, repo.Create(ctx, i))
	return i
}

func TestNew(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	tf := observability.NewNoopTracerFactory(t)
	expected := &repository{tracer: tf.Infra(), db: testDB}

	assert.Equal(t, expected, New(testDB, tf))
}

func Test_repository_Create(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	repo := &repository{tracer: observability.NewMockInfraLayerTracer(t), db: testDB}
	txm := testkit.NewTestTransactionRunner(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("問い合わせを登録し利用者から引ける", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				i := createInquiry(ctx, t, repo, seedUserA)

				got, err := repo.FindActiveByUserID(ctx, i.UserID())
				require.NoError(t, err)
				assert.Equal(t, i.ID(), got.ID())
				assert.Equal(t, i.UserID(), got.UserID())
			})
		})

		t.Run("作成日時と更新日時はDBの既定値が入る", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				i := createInquiry(ctx, t, repo, seedUserB)

				got, err := repo.FindByID(ctx, i.ID())
				require.NoError(t, err)
				assert.False(t, got.CreatedAt().IsZero())
				assert.False(t, got.UpdatedAt().IsZero())
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("同じ利用者の問い合わせが既にあればConflictを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				first := createInquiry(ctx, t, repo, seedUserC)
				second, err := domaininquiry.New(
					mustNewUUID(t), domaininquiry.Attributes{UserID: first.UserID()},
				)
				require.NoError(t, err)

				require.ErrorIs(t, repo.Create(ctx, second), apperror.ErrConflict)
			})
		})
	})
}

func Test_repository_FindByID(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	repo := &repository{tracer: observability.NewMockInfraLayerTracer(t), db: testDB}
	txm := testkit.NewTestTransactionRunner(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("IDから問い合わせを再構築して返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				i := createInquiry(ctx, t, repo, seedUserD)

				got, err := repo.FindByID(ctx, i.ID())
				require.NoError(t, err)
				assert.Equal(t, i.ID(), got.ID())
				assert.Equal(t, i.UserID(), got.UserID())
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("存在しないIDはNotFoundを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				_, err := repo.FindByID(ctx, mustNewUUID(t))
				require.ErrorIs(t, err, apperror.ErrNotFound)
			})
		})
	})
}

func Test_repository_FindActiveByUserID(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	repo := &repository{tracer: observability.NewMockInfraLayerTracer(t), db: testDB}
	txm := testkit.NewTestTransactionRunner(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("利用者から問い合わせを再構築して返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				i := createInquiry(ctx, t, repo, seedUserE)

				got, err := repo.FindActiveByUserID(ctx, i.UserID())
				require.NoError(t, err)
				assert.Equal(t, i.ID(), got.ID())
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("問い合わせを持たない利用者はNotFoundを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				_, err := repo.FindActiveByUserID(ctx, mustNewUUID(t))
				require.ErrorIs(t, err, apperror.ErrNotFound)
			})
		})
	})
}

func Test_repository_Touch(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	repo := &repository{tracer: observability.NewMockInfraLayerTracer(t), db: testDB}
	txm := testkit.NewTestTransactionRunner(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("更新日時を指定した時刻へ進める", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				i := createInquiry(ctx, t, repo, seedUserF)
				stored, err := repo.FindByID(ctx, i.ID())
				require.NoError(t, err)
				now := stored.UpdatedAt().Add(time.Hour).Truncate(time.Microsecond)

				require.NoError(t, repo.Touch(ctx, i.ID(), now))

				got, err := repo.FindByID(ctx, i.ID())
				require.NoError(t, err)
				assert.Equal(t, now, got.UpdatedAt().UTC())
			})
		})

		t.Run("存在しないIDでもエラーにしない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				require.NoError(t, repo.Touch(ctx, mustNewUUID(t), time.Now().UTC()))
			})
		})
	})
}

func Test_repository_ListForOperator(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	repo := &repository{tracer: observability.NewMockInfraLayerTracer(t), db: testDB}
	txm := testkit.NewTestTransactionRunner(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("更新日時の新しい順に先頭ページを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				older := createInquiry(ctx, t, repo, seedUserG)
				base := time.Now().UTC().Truncate(time.Microsecond)
				require.NoError(t, repo.Touch(ctx, older.ID(), base.Add(-2*time.Hour)))

				got, err := repo.ListForOperator(ctx, domaininquiry.ListParams{Limit: 100})
				require.NoError(t, err)

				require.NotEmpty(t, got)
				for idx := 1; idx < len(got); idx++ {
					assert.False(t, got[idx-1].UpdatedAt().Before(got[idx].UpdatedAt()))
				}
			})
		})

		t.Run("cursorを渡すとその位置より後ろだけを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				first, err := repo.ListForOperator(ctx, domaininquiry.ListParams{Limit: 1})
				require.NoError(t, err)
				if len(first) == 0 {
					t.Skip("問い合わせが 1 件も無いため cursor を組めない")
				}
				cursor := &domaininquiry.Cursor{UpdatedAt: first[0].UpdatedAt(), ID: first[0].ID()}

				got, err := repo.ListForOperator(ctx, domaininquiry.ListParams{Cursor: cursor, Limit: 100})
				require.NoError(t, err)

				for _, i := range got {
					assert.False(t, i.UpdatedAt().After(cursor.UpdatedAt))
					assert.NotEqual(t, cursor.ID, i.ID())
				}
			})
		})

		t.Run("上限件数を超えて返さない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				got, err := repo.ListForOperator(ctx, domaininquiry.ListParams{Limit: 1})
				require.NoError(t, err)
				assert.LessOrEqual(t, len(got), 1)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("int32へ収まらない上限件数はエラーを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				_, err := repo.ListForOperator(ctx, domaininquiry.ListParams{Limit: math.MaxInt32 + 1})
				require.Error(t, err)
			})
		})
	})
}
