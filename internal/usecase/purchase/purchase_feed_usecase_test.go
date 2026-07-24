package purchase

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	domainpurchase "go-boilerplate/internal/domain/purchase"
	mock_purchase "go-boilerplate/internal/domain/purchase/mock"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/tools/paging"
	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func Test_usecase_GetPurchases(t *testing.T) {
	t.Parallel()

	base := time.Date(2099, time.January, 1, 0, 0, 0, 0, time.UTC)

	feedItem := func(t *testing.T, salt string, orderedAt time.Time) domainpurchase.FeedItem {
		t.Helper()
		return domainpurchase.FeedItem{
			Code:        "code-" + salt,
			TotalAmount: 176500,
			StatusName:  "完了",
			OrderedAt:   orderedAt,
			ID:          uuid.NewTestFromSalt(t, salt),
		}
	}

	firstCursor := func(t *testing.T, first int) *paging.Cursor {
		t.Helper()
		c, err := paging.NewCursor(nil, &first)
		require.NoError(t, err)
		return c
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("取得件数がlimitを超えるとき末尾を切り詰めnextCursorを返す", func(t *testing.T) {
			t.Parallel()

			lt := observability.NewMockUsecaseLayerTracer(t)
			repo := mock_purchase.NewMockRepository(gomock.NewController(t))

			// limit=2 のため Repository へは 3 件（limit+1）要求され、3 件返る = 次ページあり。
			items := []domainpurchase.FeedItem{
				feedItem(t, "a", base),
				feedItem(t, "b", base.Add(-time.Hour)),
				feedItem(t, "c", base.Add(-2*time.Hour)),
			}
			repo.EXPECT().FindFeedByUserID(gomock.Any(), gomock.Any(), gomock.Any()).Return(items, nil)

			u := &usecase{tracer: lt, repo: repo}
			got, err := u.GetPurchases(context.Background(), uuid.NewTestFromSalt(t, "user"), firstCursor(t, 2))
			require.NoError(t, err)
			require.Len(t, got.Items, 2)
			// FeedItem の全フィールドが漏れなく PurchaseSummaryView へ写像されることを固定する。
			assert.Equal(t, "code-a", got.Items[0].Code)
			assert.Equal(t, "完了", got.Items[0].Status)
			assert.Equal(t, items[0].TotalAmount, got.Items[0].TotalAmount)
			assert.True(t, items[0].OrderedAt.Equal(got.Items[0].OrderedAt))

			// nextCursor は切り詰め後の末尾行（items[1]）の (ordered_at, id) から符号化される。
			require.NotNil(t, got.NextCursor)
			expected := paging.EncodeCursor(items[1].OrderedAt.Format(time.RFC3339Nano), items[1].ID.String())
			assert.Equal(t, expected, *got.NextCursor)
		})

		t.Run("取得件数がlimit未満のとき末尾ページとしてnextCursorがnilになる", func(t *testing.T) {
			t.Parallel()

			lt := observability.NewMockUsecaseLayerTracer(t)
			repo := mock_purchase.NewMockRepository(gomock.NewController(t))

			items := []domainpurchase.FeedItem{feedItem(t, "a", base)}
			repo.EXPECT().FindFeedByUserID(gomock.Any(), gomock.Any(), gomock.Any()).Return(items, nil)

			u := &usecase{tracer: lt, repo: repo}
			got, err := u.GetPurchases(context.Background(), uuid.NewTestFromSalt(t, "user"), firstCursor(t, 2))
			require.NoError(t, err)
			require.Len(t, got.Items, 1)
			assert.Nil(t, got.NextCursor)
		})

		t.Run("取得件数がlimitとちょうど一致するとき末尾ページになる", func(t *testing.T) {
			t.Parallel()

			lt := observability.NewMockUsecaseLayerTracer(t)
			repo := mock_purchase.NewMockRepository(gomock.NewController(t))

			// limit=2 に対し Repository が limit+1(=3) 未満のちょうど 2 件を返す = 次ページ無し。
			// hasNext 判定が `len > limit`（`>=` ではない）であることの境界検証。
			items := []domainpurchase.FeedItem{feedItem(t, "a", base), feedItem(t, "b", base.Add(-time.Hour))}
			repo.EXPECT().FindFeedByUserID(gomock.Any(), gomock.Any(), gomock.Any()).Return(items, nil)

			u := &usecase{tracer: lt, repo: repo}
			got, err := u.GetPurchases(context.Background(), uuid.NewTestFromSalt(t, "user"), firstCursor(t, 2))
			require.NoError(t, err)
			require.Len(t, got.Items, 2)
			assert.Nil(t, got.NextCursor)
		})

		t.Run("購入がゼロのとき空配列かつnextCursorがnilになる", func(t *testing.T) {
			t.Parallel()

			lt := observability.NewMockUsecaseLayerTracer(t)
			repo := mock_purchase.NewMockRepository(gomock.NewController(t))

			repo.EXPECT().FindFeedByUserID(gomock.Any(), gomock.Any(), gomock.Any()).Return([]domainpurchase.FeedItem{}, nil)

			u := &usecase{tracer: lt, repo: repo}
			got, err := u.GetPurchases(context.Background(), uuid.NewTestFromSalt(t, "user"), firstCursor(t, 2))
			require.NoError(t, err)
			assert.Empty(t, got.Items)
			assert.Nil(t, got.NextCursor)
		})

		t.Run("afterカーソルが復号されuserIDとkeyset境界がRepositoryへ渡る", func(t *testing.T) {
			t.Parallel()

			lt := observability.NewMockUsecaseLayerTracer(t)
			repo := mock_purchase.NewMockRepository(gomock.NewController(t))

			userID := uuid.NewTestFromSalt(t, "user")
			boundaryID := uuid.NewTestFromSalt(t, "boundary")
			after := paging.EncodeCursor(base.Format(time.RFC3339Nano), boundaryID.String())
			first := 2
			cursor, err := paging.NewCursor(&after, &first)
			require.NoError(t, err)

			var captured domainpurchase.ListFeedParams
			var capturedUserID uuid.UUID
			repo.EXPECT().FindFeedByUserID(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, uid uuid.UUID, params domainpurchase.ListFeedParams) ([]domainpurchase.FeedItem, error) {
					capturedUserID = uid
					captured = params
					return []domainpurchase.FeedItem{}, nil
				},
			)

			u := &usecase{tracer: lt, repo: repo}
			_, err = u.GetPurchases(context.Background(), userID, cursor)
			require.NoError(t, err)

			assert.Equal(t, userID, capturedUserID)
			assert.Equal(t, int32(3), captured.Limit) // limit(2) + 1
			require.NotNil(t, captured.AfterOrderedAt)
			require.NotNil(t, captured.AfterID)
			assert.True(t, base.Equal(*captured.AfterOrderedAt))
			assert.Equal(t, boundaryID, *captured.AfterID)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("cursorがnilのときErrInvalidArgumentを返しRepositoryを呼ばない", func(t *testing.T) {
			t.Parallel()

			lt := observability.NewMockUsecaseLayerTracer(t)
			repo := mock_purchase.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().FindFeedByUserID(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			u := &usecase{tracer: lt, repo: repo}
			got, err := u.GetPurchases(context.Background(), uuid.NewTestFromSalt(t, "user"), nil)
			assert.Nil(t, got)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("cursorの復号に失敗したときErrInvalidArgumentを返しRepositoryを呼ばない", func(t *testing.T) {
			t.Parallel()

			lt := observability.NewMockUsecaseLayerTracer(t)
			repo := mock_purchase.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().FindFeedByUserID(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			// キーが 1 個（ordered_at, id の 2 個でない）= 復号エラー。
			after := paging.EncodeCursor("only-one-key")
			first := 2
			cursor, err := paging.NewCursor(&after, &first)
			require.NoError(t, err)

			u := &usecase{tracer: lt, repo: repo}
			got, err := u.GetPurchases(context.Background(), uuid.NewTestFromSalt(t, "user"), cursor)
			assert.Nil(t, got)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("Repositoryのエラーが伝播する", func(t *testing.T) {
			t.Parallel()

			lt := observability.NewMockUsecaseLayerTracer(t)
			repo := mock_purchase.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().FindFeedByUserID(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, apperror.ErrInternal)

			u := &usecase{tracer: lt, repo: repo}
			got, err := u.GetPurchases(context.Background(), uuid.NewTestFromSalt(t, "user"), firstCursor(t, 2))
			assert.Nil(t, got)
			require.ErrorIs(t, err, apperror.ErrInternal)
		})
	})
}
