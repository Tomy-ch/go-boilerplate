package inquirymessage

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/apperror"
	domaininquiry "go-boilerplate/internal/domain/inquiry"
	domainmessage "go-boilerplate/internal/domain/inquirymessage"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	inquiryrepo "go-boilerplate/internal/infrastructure/rdb/repository/inquiry"
	"go-boilerplate/internal/infrastructure/rdb/sqlc/gen"
	"go-boilerplate/internal/infrastructure/rdb/testkit"
	"go-boilerplate/internal/observability"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/uuid"
)

// takeSeedUser は、seed 済みの利用者を ID 順で index 番目に選びます。
//
// seed の SQL から ID を書き写すと、その利用者が実際に投入されているかは環境によって変わり、
// 外部キー違反で落ちます。並び順で取ることで、どの環境でも実在する利用者だけを使います。
// 問い合わせは利用者ごとに 1 件しか持てないため、並行するケースには別の index を渡します。
func takeSeedUser(ctx context.Context, t *testing.T, testDB driver.DatabaseDriver, index int) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := driver.New(ctx, testDB).
		QueryRow(ctx, "SELECT id FROM users ORDER BY id LIMIT 1 OFFSET $1", index).
		Scan(&id)
	require.NoError(t, err, "seed 済みの利用者が %d 人未満です", index+1)
	return id
}

func mustNewUUID(t *testing.T) uuid.UUID {
	t.Helper()
	id, err := uuid.New()
	require.NoError(t, err)
	return id
}

// createInquiry は、メッセージの FK を満たす問い合わせを 1 件登録し、その ID を返します。
func createInquiry(ctx context.Context, t *testing.T, testDB driver.DatabaseDriver, userIndex int) (uuid.UUID, uuid.UUID) {
	t.Helper()
	repo := inquiryrepo.New(testDB, observability.NewNoopTracerFactory(t))
	userID := takeSeedUser(ctx, t, testDB, userIndex)
	i, err := domaininquiry.New(mustNewUUID(t), domaininquiry.Attributes{UserID: userID})
	require.NoError(t, err)
	require.NoError(t, repo.Create(ctx, i))
	return i.ID(), userID
}

// newMessage は、指定した位置のメッセージを組み立てます。
func newMessage(t *testing.T, inquiryID uuid.UUID, kind domainmessage.AuthorKind, subject uuid.UUID, seq int64) *domainmessage.Message {
	t.Helper()
	author, err := domainmessage.NewAuthor(kind, subject)
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
				inquiryID, userID := createInquiry(ctx, t, testDB, 0)
				m := newMessage(t, inquiryID, domainmessage.AuthorKindUser, userID, 1)

				require.NoError(t, repo.Create(ctx, m))

				got, err := repo.ListByInquiry(ctx, inquiryID, domainmessage.HistoryParams{UpToSequence: 1, Limit: 10})
				require.NoError(t, err)
				require.Len(t, got, 1)
				assert.Equal(t, m.ID(), got[0].ID())
				assert.Equal(t, domainmessage.AuthorKindUser, got[0].Author().Kind())
				assert.Equal(t, userID, got[0].Author().SubjectID())
				assert.Equal(t, int64(1), got[0].Sequence())
			})
		})

		t.Run("回答者のメッセージも種別を保って往復する", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				inquiryID, userID := createInquiry(ctx, t, testDB, 1)
				m := newMessage(t, inquiryID, domainmessage.AuthorKindOperator, userID, 1)

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
				inquiryID, userID := createInquiry(ctx, t, testDB, 2)
				require.NoError(t, repo.Create(ctx, newMessage(t, inquiryID, domainmessage.AuthorKindUser, userID, 1)))

				err := repo.Create(ctx, newMessage(t, inquiryID, domainmessage.AuthorKindUser, userID, 1))
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
				inquiryID, userID := createInquiry(ctx, t, testDB, 3)
				for seq := int64(1); seq <= 3; seq++ {
					require.NoError(t, repo.Create(ctx, newMessage(t, inquiryID, domainmessage.AuthorKindUser, userID, seq)))
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
				inquiryID, userID := createInquiry(ctx, t, testDB, 4)
				for seq := int64(1); seq <= 3; seq++ {
					require.NoError(t, repo.Create(ctx, newMessage(t, inquiryID, domainmessage.AuthorKindUser, userID, seq)))
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

func Test_reconstruct(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("送り手を列の2つ組から値オブジェクトへ組み直す", func(t *testing.T) {
			t.Parallel()
			id, inquiryID, subjectID := mustNewUUID(t), mustNewUUID(t), mustNewUUID(t)
			createdAt := time.Date(2026, time.September, 1, 10, 0, 0, 0, time.UTC)

			got, err := reconstruct(gen.InquiryMessages{
				ID:              id,
				InquiryID:       inquiryID,
				AuthorKind:      "operator",
				AuthorSubjectID: subjectID,
				Body:            "本文",
				StreamSequence:  3,
				CreatedAt:       createdAt,
			})

			require.NoError(t, err)
			assert.Equal(t, id, got.ID())
			assert.Equal(t, inquiryID, got.InquiryID())
			assert.Equal(t, domainmessage.AuthorKindOperator, got.Author().Kind())
			assert.Equal(t, subjectID, got.Author().SubjectID())
			assert.Equal(t, int64(3), got.Sequence())
			assert.Equal(t, createdAt, got.CreatedAt())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		// 表の CHECK 制約と NOT NULL がある限り到達しないが、種別の検証はこの関数が
		// 担う境界なので、壊れた行を渡したときに集約を組み立てないことを固定する。
		t.Run("既知でない種別の行はErrInvalidAuthorKindを返す", func(t *testing.T) {
			t.Parallel()

			_, err := reconstruct(gen.InquiryMessages{
				ID: mustNewUUID(t), InquiryID: mustNewUUID(t),
				AuthorKind: "admin", AuthorSubjectID: mustNewUUID(t),
				Body: "本文", StreamSequence: 1,
			})

			require.ErrorIs(t, err, domainmessage.ErrInvalidAuthorKind)
		})

		t.Run("送り手が未設定の行はErrInvalidAuthorSubjectを返す", func(t *testing.T) {
			t.Parallel()

			_, err := reconstruct(gen.InquiryMessages{
				ID: mustNewUUID(t), InquiryID: mustNewUUID(t),
				AuthorKind: "user", Body: "本文", StreamSequence: 1,
			})

			require.ErrorIs(t, err, domainmessage.ErrInvalidAuthorSubject)
		})

		t.Run("位置が0の行はErrInvalidSequenceを返す", func(t *testing.T) {
			t.Parallel()

			_, err := reconstruct(gen.InquiryMessages{
				ID: mustNewUUID(t), InquiryID: mustNewUUID(t),
				AuthorKind: "user", AuthorSubjectID: mustNewUUID(t), Body: "本文",
			})

			require.ErrorIs(t, err, domainmessage.ErrInvalidSequence)
		})
	})
}
