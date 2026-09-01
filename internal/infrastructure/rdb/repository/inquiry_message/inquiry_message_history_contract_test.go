package inquirymessage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainmessage "go-boilerplate/internal/domain/inquirymessage"
	realtimesq "go-boilerplate/internal/infrastructure/rdb/system_cqrs/realtime"
	"go-boilerplate/internal/infrastructure/rdb/testkit"
	"go-boilerplate/internal/observability"
	rt "go-boilerplate/internal/usecase/boundary/realtime"
	"go-boilerplate/pkg/ptr"
)

// 履歴の同時性契約が使う seed 利用者。inquiries.user_id は UNIQUE のため他のケースと重ねません。
const (
	seedRaceUserA = "058026a6-82d9-4538-9f45-e18a3cd8c99a"
	seedRaceUserB = "0775fe11-df27-4488-92de-018b4fae66b1"
)

// TestHistoryCursorContract は、履歴の取得と購読の開始の間に追加された 1 通が失われないことを、
// 実データベース上で確かめます。
//
// 親受入基準 5（History 取得と SSE 接続の間の event を取りこぼさない）のうち、**履歴側の契約**を
// 対象にします。購読側（cursor より後ろを再生すること）は機構の責務で、replay の試験が別に持ちます。
// ここで固定するのは「現在位置を先に読み、その位置を上限にして読む」という手順が、
// 上限より後ろの 1 通を履歴へ混ぜず、かつ client がその 1 通へ到達できる位置を返すことです。
func TestHistoryCursorContract(t *testing.T) {
	t.Parallel()

	testDB := testkit.NewTestDB(t)
	repo := &repository{tracer: observability.NewMockInfraLayerTracer(t), db: testDB}
	sequences := realtimesq.NewSequenceAllocator(testDB, observability.NewNoopTracerFactory(t))
	txm := testkit.NewTestTransactionRunner(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("現在位置を読んだ後に追加された1通は履歴へ混ざらず位置で辿れる", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				inquiryID := createInquiry(ctx, t, testDB, seedRaceUserA)
				stream := rt.StreamID(inquiryID.String())

				// 履歴に載るべき 2 通。
				for range 2 {
					seq, err := sequences.Allocate(ctx, stream)
					require.NoError(t, err)
					require.NoError(t, repo.Create(ctx, newMessage(
						t, inquiryID, domainmessage.AuthorKindUser, seedRaceUserA, int64(seq),
					)))
				}

				// 履歴はまずここで現在位置を読む。
				cursor, ok, err := sequences.Current(ctx, stream)
				require.NoError(t, err)
				require.True(t, ok)
				require.Equal(t, rt.Sequence(2), cursor)

				// 現在位置を読んだ後、メッセージを読む前に 1 通増える（取りこぼしが起きるとしたらこの窓）。
				raced, err := sequences.Allocate(ctx, stream)
				require.NoError(t, err)
				require.NoError(t, repo.Create(ctx, newMessage(
					t, inquiryID, domainmessage.AuthorKindUser, seedRaceUserA, int64(raced),
				)))

				got, err := repo.ListByInquiry(ctx, inquiryID, domainmessage.HistoryParams{
					UpToSequence: int64(cursor), Limit: 100,
				})
				require.NoError(t, err)

				// 履歴は現在位置までで閉じており、窓の中で増えた 1 通を含まない。
				require.Len(t, got, 2)
				assert.Equal(t, int64(1), got[0].Sequence())
				assert.Equal(t, int64(2), got[1].Sequence())

				// その 1 通は現在位置より後ろにあるため、client が現在位置から購読を始めれば届く。
				assert.Greater(t, int64(raced), int64(cursor))

				resumed, err := repo.ListByInquiry(ctx, inquiryID, domainmessage.HistoryParams{
					AfterSequence: ptr.To(int64(cursor)), UpToSequence: int64(raced), Limit: 100,
				})
				require.NoError(t, err)
				require.Len(t, resumed, 1)
				assert.Equal(t, int64(raced), resumed[0].Sequence())
			})
		})

		t.Run("採番は欠番なく単調に進む", func(t *testing.T) {
			t.Parallel()

			txm.WithinTx(func(ctx context.Context) {
				inquiryID := createInquiry(ctx, t, testDB, seedRaceUserB)
				stream := rt.StreamID(inquiryID.String())

				allocated := make([]int64, 0, 3)
				for range 3 {
					seq, err := sequences.Allocate(ctx, stream)
					require.NoError(t, err)
					allocated = append(allocated, int64(seq))
				}

				assert.Equal(t, []int64{1, 2, 3}, allocated)
			})
		})
	})
}
