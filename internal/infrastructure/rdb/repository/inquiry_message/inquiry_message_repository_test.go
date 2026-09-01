package inquirymessage

import (
	"context"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/apperror"
	domaininquiry "go-boilerplate/internal/domain/inquiry"
	domainmessage "go-boilerplate/internal/domain/inquirymessage"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	inquiryrepo "go-boilerplate/internal/infrastructure/rdb/repository/inquiry"
	"go-boilerplate/internal/infrastructure/rdb/testkit"
	"go-boilerplate/internal/observability"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/uuid"
)

// 既存 seed のユーザー。inquiries.user_id は UNIQUE のため、並行するケースは別々の利用者を使います
// （問い合わせ側のテストが使う利用者とも重ねません）。
const (
	seedUserA    = "101caa1e-84e7-4ceb-9108-50d40b6be1a3"
	seedUserB    = "c23845a3-1bd6-5cc9-9aec-c6e824c65a17"
	seedUserC    = "65ecbae0-cab1-57c0-9cd0-34699624342e"
	seedUserD    = "08787d6f-9a19-46ad-aaa1-da3e369c343b"
	seedUserE    = "3c6b7ebc-983e-518a-bbb9-4e287e18c84e"
	seedOperator = "d647fc85-ff46-4530-88cb-198f4a68a9d7"
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

// createInquiry は、メッセージの FK を満たす問い合わせを 1 件登録し、その ID を返します。
func createInquiry(ctx context.Context, t *testing.T, testDB driver.DatabaseDriver, seedUser string) uuid.UUID {
	t.Helper()
	repo := inquiryrepo.New(testDB, observability.NewNoopTracerFactory(t))
	i, err := domaininquiry.New(mustNewUUID(t), domaininquiry.Attributes{UserID: mustParse(t, seedUser)})
	require.NoError(t, err)
	require.NoError(t, repo.Create(ctx, i))
	return i.ID()
}

// newMessage は、指定した位置のメッセージを組み立てます。
func newMessage(t *testing.T, inquiryID uuid.UUID, kind domainmessage.AuthorKind, subject string, seq int64) *domainmessage.Message {
	t.Helper()
	author, err := domainmessage.NewAuthor(kind, mustParse(t, subject))
	require.NoError(t, err)
	m, err := domainmessage.New(mustNewUUID(t), domainmessage.Attributes{
		InquiryID: inquiryID,
		Author:    author,
		Body:      "本文",
		Sequence:  seq,
	})
	require.NoError(t, err)
	return m
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

		t.Run("メッセージを登録し送り手ごと再構築できる", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				inquiryID := createInquiry(ctx, t, testDB, seedUserA)
				m := newMessage(t, inquiryID, domainmessage.AuthorKindUser, seedUserA, 1)

				require.NoError(t, repo.Create(ctx, m))

				got, err := repo.ListByInquiry(ctx, inquiryID, domainmessage.HistoryParams{UpToSequence: 1, Limit: 10})
				require.NoError(t, err)
				require.Len(t, got, 1)
				assert.Equal(t, m.ID(), got[0].ID())
				assert.Equal(t, domainmessage.AuthorKindUser, got[0].Author().Kind())
				assert.Equal(t, mustParse(t, seedUserA), got[0].Author().SubjectID())
				assert.Equal(t, int64(1), got[0].Sequence())
			})
		})

		t.Run("回答者のメッセージも種別を保って往復する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				inquiryID := createInquiry(ctx, t, testDB, seedUserB)
				m := newMessage(t, inquiryID, domainmessage.AuthorKindOperator, seedOperator, 1)

				require.NoError(t, repo.Create(ctx, m))

				got, err := repo.ListByInquiry(ctx, inquiryID, domainmessage.HistoryParams{UpToSequence: 1, Limit: 10})
				require.NoError(t, err)
				require.Len(t, got, 1)
				assert.Equal(t, domainmessage.AuthorKindOperator, got[0].Author().Kind())
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("同じ問い合わせに同じ位置を二重登録するとConflictを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				inquiryID := createInquiry(ctx, t, testDB, seedUserC)
				require.NoError(t, repo.Create(ctx, newMessage(t, inquiryID, domainmessage.AuthorKindUser, seedUserC, 1)))

				err := repo.Create(ctx, newMessage(t, inquiryID, domainmessage.AuthorKindUser, seedUserC, 1))
				require.ErrorIs(t, err, apperror.ErrConflict)
			})
		})
	})
}

func Test_repository_ListByInquiry(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	repo := &repository{tracer: observability.NewMockInfraLayerTracer(t), db: testDB}
	txm := testkit.NewTestTransactionRunner(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("位置の昇順で返し上限より後ろは含めない", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				inquiryID := createInquiry(ctx, t, testDB, seedUserD)
				for seq := int64(1); seq <= 3; seq++ {
					require.NoError(t, repo.Create(ctx, newMessage(t, inquiryID, domainmessage.AuthorKindUser, seedUserD, seq)))
				}

				got, err := repo.ListByInquiry(ctx, inquiryID, domainmessage.HistoryParams{UpToSequence: 2, Limit: 10})
				require.NoError(t, err)
				require.Len(t, got, 2)
				assert.Equal(t, int64(1), got[0].Sequence())
				assert.Equal(t, int64(2), got[1].Sequence())
			})
		})

		t.Run("開始位置より後ろだけを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				inquiryID := createInquiry(ctx, t, testDB, seedUserE)
				for seq := int64(1); seq <= 3; seq++ {
					require.NoError(t, repo.Create(ctx, newMessage(t, inquiryID, domainmessage.AuthorKindUser, seedUserE, seq)))
				}

				got, err := repo.ListByInquiry(ctx, inquiryID, domainmessage.HistoryParams{
					AfterSequence: ptr.To(int64(1)), UpToSequence: 3, Limit: 10,
				})
				require.NoError(t, err)
				require.Len(t, got, 2)
				assert.Equal(t, int64(2), got[0].Sequence())
			})
		})

		t.Run("メッセージが無ければ空を返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				got, err := repo.ListByInquiry(ctx, mustNewUUID(t), domainmessage.HistoryParams{UpToSequence: 10, Limit: 10})
				require.NoError(t, err)
				assert.Empty(t, got)
			})
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("int32へ収まらない上限件数はエラーを返す", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				_, err := repo.ListByInquiry(ctx, mustNewUUID(t), domainmessage.HistoryParams{
					UpToSequence: 1, Limit: math.MaxInt32 + 1,
				})
				require.Error(t, err)
			})
		})
	})
}
