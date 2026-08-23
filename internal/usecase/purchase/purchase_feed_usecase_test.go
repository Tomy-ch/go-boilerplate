package purchase

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	domainpurchase "go-boilerplate/internal/domain/purchase"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/internal/usecase/boundary/authz"
	mock_authz "go-boilerplate/internal/usecase/boundary/authz/mock"
	"go-boilerplate/internal/usecase/purchase/query"
	mock_query "go-boilerplate/internal/usecase/purchase/query/mock"
	"go-boilerplate/internal/usecase/tools/paging"
	"go-boilerplate/internal/usecase/tools/timewindow"
	"go-boilerplate/pkg/safecast"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func Test_usecase_GetPurchases(t *testing.T) {
	t.Parallel()

	base := time.Date(2099, time.January, 1, 0, 0, 0, 0, time.UTC)

	feedItem := func(t *testing.T, salt string, orderedAt time.Time) query.PurchaseFeedReadModel {
		t.Helper()
		return query.PurchaseFeedReadModel{
			Code:          "code-" + salt,
			TotalAmount:   176500,
			StatusID:      uuidtestkit.NewTestFromSalt(t, salt+"_status"),
			StatusCode:    domainpurchase.StatusCompleted.Code(),
			StatusName:    "完了",
			FirstItemName: "ワイヤレスイヤホン",
			ItemCount:     3,
			OrderedAt:     orderedAt,
			ID:            uuidtestkit.NewTestFromSalt(t, salt),
		}
	}

	firstCursor := func(t *testing.T, first int) *paging.Cursor {
		t.Helper()
		c, err := paging.NewCursor(nil, &first)
		require.NoError(t, err)
		return c
	}

	statusCodeOf := func(t *testing.T, status domainpurchase.Status) int16 {
		t.Helper()
		code, err := safecast.IntToInt16(status.Code())
		require.NoError(t, err)
		return code
	}

	authnOf := func(t *testing.T, userID uuid.UUID) *auth.Authn {
		t.Helper()
		a, err := auth.New("feed-uc-subject", "feed-uc-issuer", nil, nil)
		require.NoError(t, err)
		resolved, err := a.WithUserID(userID)
		require.NoError(t, err)
		return resolved
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("取得件数がlimitを超えるとき末尾を切り詰めnextCursorを返す", func(t *testing.T) {
			t.Parallel()

			lt := observability.NewMockUsecaseLayerTracer(t)
			feedQS := mock_query.NewMockPurchaseFeedQueryService(gomock.NewController(t))

			// limit=2 のため Repository へは 3 件（limit+1）要求され、3 件返る = 次ページあり。
			items := []query.PurchaseFeedReadModel{
				feedItem(t, "a", base),
				feedItem(t, "b", base.Add(-time.Hour)),
				feedItem(t, "c", base.Add(-2*time.Hour)),
			}
			feedQS.EXPECT().FindFeedByUserID(gomock.Any(), gomock.Any(), gomock.Any()).Return(items, nil)

			u := &usecase{tracer: lt, feedQS: feedQS}
			got, err := u.GetPurchases(
				context.Background(),
				authnOf(t, uuidtestkit.NewTestFromSalt(t, "user")),
				ListPurchasesParams{Cursor: firstCursor(t, 2)},
			)
			require.NoError(t, err)
			require.Len(t, got.Items, 2)
			// FeedItem の全フィールドが漏れなく PurchaseSummaryView へ写像されることを固定する。
			assert.Equal(t, "code-a", got.Items[0].Code)
			assert.Equal(t, items[0].StatusID, got.Items[0].StatusID)
			assert.Equal(t, "完了", got.Items[0].StatusName)
			assert.Equal(t, items[0].StatusCode, got.Items[0].StatusCode)
			assert.Equal(t, items[0].TotalAmount, got.Items[0].TotalAmount)
			assert.Equal(t, items[0].FirstItemName, got.Items[0].FirstItemName)
			assert.Equal(t, items[0].ItemCount, got.Items[0].ItemCount)
			assert.True(t, items[0].OrderedAt.Equal(got.Items[0].OrderedAt))

			// nextCursor は切り詰め後の末尾行（items[1]）の (ordered_at, id) から符号化される。
			require.NotNil(t, got.NextCursor)
			expected := paging.EncodeCursor(items[1].OrderedAt.Format(time.RFC3339Nano), items[1].ID.String())
			assert.Equal(t, expected, *got.NextCursor)
		})

		t.Run("取得件数がlimit未満のとき末尾ページとしてnextCursorがnilになる", func(t *testing.T) {
			t.Parallel()

			lt := observability.NewMockUsecaseLayerTracer(t)
			feedQS := mock_query.NewMockPurchaseFeedQueryService(gomock.NewController(t))

			items := []query.PurchaseFeedReadModel{feedItem(t, "a", base)}
			feedQS.EXPECT().FindFeedByUserID(gomock.Any(), gomock.Any(), gomock.Any()).Return(items, nil)

			u := &usecase{tracer: lt, feedQS: feedQS}
			got, err := u.GetPurchases(
				context.Background(),
				authnOf(t, uuidtestkit.NewTestFromSalt(t, "user")),
				ListPurchasesParams{Cursor: firstCursor(t, 2)},
			)
			require.NoError(t, err)
			require.Len(t, got.Items, 1)
			assert.Nil(t, got.NextCursor)
		})

		t.Run("取得件数がlimitとちょうど一致するとき末尾ページになる", func(t *testing.T) {
			t.Parallel()

			lt := observability.NewMockUsecaseLayerTracer(t)
			feedQS := mock_query.NewMockPurchaseFeedQueryService(gomock.NewController(t))

			// limit=2 に対し Repository が limit+1(=3) 未満のちょうど 2 件を返す = 次ページ無し。
			// hasNext 判定が `len > limit`（`>=` ではない）であることの境界検証。
			items := []query.PurchaseFeedReadModel{feedItem(t, "a", base), feedItem(t, "b", base.Add(-time.Hour))}
			feedQS.EXPECT().FindFeedByUserID(gomock.Any(), gomock.Any(), gomock.Any()).Return(items, nil)

			u := &usecase{tracer: lt, feedQS: feedQS}
			got, err := u.GetPurchases(
				context.Background(),
				authnOf(t, uuidtestkit.NewTestFromSalt(t, "user")),
				ListPurchasesParams{Cursor: firstCursor(t, 2)},
			)
			require.NoError(t, err)
			require.Len(t, got.Items, 2)
			assert.Nil(t, got.NextCursor)
		})

		t.Run("購入がゼロのとき空配列かつnextCursorがnilになる", func(t *testing.T) {
			t.Parallel()

			lt := observability.NewMockUsecaseLayerTracer(t)
			feedQS := mock_query.NewMockPurchaseFeedQueryService(gomock.NewController(t))

			feedQS.EXPECT().FindFeedByUserID(gomock.Any(), gomock.Any(), gomock.Any()).Return([]query.PurchaseFeedReadModel{}, nil)

			u := &usecase{tracer: lt, feedQS: feedQS}
			got, err := u.GetPurchases(
				context.Background(),
				authnOf(t, uuidtestkit.NewTestFromSalt(t, "user")),
				ListPurchasesParams{Cursor: firstCursor(t, 2)},
			)
			require.NoError(t, err)
			assert.Empty(t, got.Items)
			assert.Nil(t, got.NextCursor)
		})

		t.Run("ゼロ値Cursor混入時もfeed末尾のpanicを避けItems空nextCursor_nilを返す", func(t *testing.T) {
			t.Parallel()

			// ゼロ値 Cursor（limit=0）では Limit()=0 かつ Repository へは limit+1=1 件要求される。
			// 1 件返ると hasNext=len(1)>0=true・切り詰めで feed が空になり、len(feed)>0 の安全弁が無いと
			// feed[len-1] が feed[-1] で panic する。安全弁により panic せず空一覧を返すことを固定する。
			lt := observability.NewMockUsecaseLayerTracer(t)
			feedQS := mock_query.NewMockPurchaseFeedQueryService(gomock.NewController(t))
			feedQS.EXPECT().FindFeedByUserID(gomock.Any(), gomock.Any(), gomock.Any()).Return(
				[]query.PurchaseFeedReadModel{feedItem(t, "a", base)}, nil)

			u := &usecase{tracer: lt, feedQS: feedQS}
			got, err := u.GetPurchases(
				context.Background(),
				authnOf(t, uuidtestkit.NewTestFromSalt(t, "user")),
				ListPurchasesParams{Cursor: &paging.Cursor{}},
			)
			require.NoError(t, err)
			assert.Empty(t, got.Items)
			assert.Nil(t, got.NextCursor)
		})

		t.Run("afterカーソルが復号されuserIDとkeyset境界がRepositoryへ渡る", func(t *testing.T) {
			t.Parallel()

			lt := observability.NewMockUsecaseLayerTracer(t)
			feedQS := mock_query.NewMockPurchaseFeedQueryService(gomock.NewController(t))

			userID := uuidtestkit.NewTestFromSalt(t, "user")
			boundaryID := uuidtestkit.NewTestFromSalt(t, "boundary")
			after := paging.EncodeCursor(base.Format(time.RFC3339Nano), boundaryID.String())
			first := 2
			cursor, err := paging.NewCursor(&after, &first)
			require.NoError(t, err)

			var captured query.ListFeedParams
			var capturedUserID uuid.UUID
			feedQS.EXPECT().FindFeedByUserID(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, uid uuid.UUID, params query.ListFeedParams) ([]query.PurchaseFeedReadModel, error) {
					capturedUserID = uid
					captured = params
					return []query.PurchaseFeedReadModel{}, nil
				},
			)

			u := &usecase{tracer: lt, feedQS: feedQS}
			_, err = u.GetPurchases(context.Background(), authnOf(t, userID), ListPurchasesParams{Cursor: cursor})
			require.NoError(t, err)

			assert.Equal(t, userID, capturedUserID)
			assert.Equal(t, int32(3), captured.Limit) // limit(2) + 1
			require.NotNil(t, captured.AfterOrderedAt)
			require.NotNil(t, captured.AfterID)
			assert.True(t, base.Equal(*captured.AfterOrderedAt))
			assert.Equal(t, boundaryID, *captured.AfterID)
			// 期間指定なしでは境界を持たない対象期間が渡り、Repository 側の期間条件が効かない。
			assert.Nil(t, captured.Window.After())
			assert.Nil(t, captured.Window.Before())
		})

		t.Run("両端を持つ対象期間がそのままRepositoryへ渡る", func(t *testing.T) {
			t.Parallel()

			lt := observability.NewMockUsecaseLayerTracer(t)
			feedQS := mock_query.NewMockPurchaseFeedQueryService(gomock.NewController(t))

			after := time.Date(2026, time.January, 21, 0, 0, 0, 0, time.UTC)
			before := time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC)
			window, err := timewindow.New(timewindow.Bounds{After: &after, Before: &before})
			require.NoError(t, err)

			var captured query.ListFeedParams
			feedQS.EXPECT().FindFeedByUserID(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, _ uuid.UUID, params query.ListFeedParams) ([]query.PurchaseFeedReadModel, error) {
					captured = params
					return []query.PurchaseFeedReadModel{}, nil
				},
			)

			u := &usecase{tracer: lt, feedQS: feedQS}
			_, err = u.GetPurchases(
				context.Background(),
				authnOf(t, uuidtestkit.NewTestFromSalt(t, "user")),
				ListPurchasesParams{Cursor: firstCursor(t, 2), Window: window},
			)
			require.NoError(t, err)

			require.NotNil(t, captured.Window.After())
			require.NotNil(t, captured.Window.Before())
			assert.True(t, after.Equal(*captured.Window.After()))
			assert.True(t, before.Equal(*captured.Window.Before()))
		})

		t.Run("片側だけの対象期間はその境界だけがQueryServiceへ渡る", func(t *testing.T) {
			t.Parallel()

			lt := observability.NewMockUsecaseLayerTracer(t)
			feedQS := mock_query.NewMockPurchaseFeedQueryService(gomock.NewController(t))

			after := time.Date(2026, time.January, 21, 0, 0, 0, 0, time.UTC)
			window, err := timewindow.New(timewindow.Bounds{After: &after})
			require.NoError(t, err)

			var captured query.ListFeedParams
			feedQS.EXPECT().FindFeedByUserID(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, _ uuid.UUID, params query.ListFeedParams) ([]query.PurchaseFeedReadModel, error) {
					captured = params
					return []query.PurchaseFeedReadModel{}, nil
				},
			)

			u := &usecase{tracer: lt, feedQS: feedQS}
			_, err = u.GetPurchases(
				context.Background(),
				authnOf(t, uuidtestkit.NewTestFromSalt(t, "user")),
				ListPurchasesParams{Cursor: firstCursor(t, 2), Window: window},
			)
			require.NoError(t, err)

			require.NotNil(t, captured.Window.After())
			assert.True(t, after.Equal(*captured.Window.After()))
			assert.Nil(t, captured.Window.Before())
		})

		t.Run("ステータスの絞り込みがそのままQueryServiceへ渡る", func(t *testing.T) {
			t.Parallel()

			lt := observability.NewMockUsecaseLayerTracer(t)
			feedQS := mock_query.NewMockPurchaseFeedQueryService(gomock.NewController(t))

			var captured query.ListFeedParams
			feedQS.EXPECT().FindFeedByUserID(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, _ uuid.UUID, params query.ListFeedParams) ([]query.PurchaseFeedReadModel, error) {
					captured = params
					return []query.PurchaseFeedReadModel{}, nil
				},
			)

			u := &usecase{tracer: lt, feedQS: feedQS}
			_, err := u.GetPurchases(
				context.Background(),
				authnOf(t, uuidtestkit.NewTestFromSalt(t, "user")),
				ListPurchasesParams{
					Cursor:      firstCursor(t, 2),
					StatusCodes: []int16{statusCodeOf(t, domainpurchase.StatusShipped)},
				},
			)
			require.NoError(t, err)

			assert.Equal(t, []int16{statusCodeOf(t, domainpurchase.StatusShipped)}, captured.StatusCodes)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("cursorがnilのときErrInvalidArgumentを返しRepositoryを呼ばない", func(t *testing.T) {
			t.Parallel()

			lt := observability.NewMockUsecaseLayerTracer(t)
			feedQS := mock_query.NewMockPurchaseFeedQueryService(gomock.NewController(t))
			feedQS.EXPECT().FindFeedByUserID(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			u := &usecase{tracer: lt, feedQS: feedQS}
			got, err := u.GetPurchases(
				context.Background(),
				authnOf(t, uuidtestkit.NewTestFromSalt(t, "user")),
				ListPurchasesParams{},
			)
			assert.Nil(t, got)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("cursorの復号に失敗したときErrInvalidArgumentを返しRepositoryを呼ばない", func(t *testing.T) {
			t.Parallel()

			lt := observability.NewMockUsecaseLayerTracer(t)
			feedQS := mock_query.NewMockPurchaseFeedQueryService(gomock.NewController(t))
			feedQS.EXPECT().FindFeedByUserID(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			// キーが 1 個（ordered_at, id の 2 個でない）= 復号エラー。
			after := paging.EncodeCursor("only-one-key")
			first := 2
			cursor, err := paging.NewCursor(&after, &first)
			require.NoError(t, err)

			u := &usecase{tracer: lt, feedQS: feedQS}
			got, err := u.GetPurchases(
				context.Background(),
				authnOf(t, uuidtestkit.NewTestFromSalt(t, "user")),
				ListPurchasesParams{Cursor: cursor},
			)
			assert.Nil(t, got)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("Repositoryのエラーが伝播する", func(t *testing.T) {
			t.Parallel()

			lt := observability.NewMockUsecaseLayerTracer(t)
			feedQS := mock_query.NewMockPurchaseFeedQueryService(gomock.NewController(t))
			feedQS.EXPECT().FindFeedByUserID(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, apperror.ErrInternal)

			u := &usecase{tracer: lt, feedQS: feedQS}
			got, err := u.GetPurchases(
				context.Background(),
				authnOf(t, uuidtestkit.NewTestFromSalt(t, "user")),
				ListPurchasesParams{Cursor: firstCursor(t, 2)},
			)
			assert.Nil(t, got)
			require.ErrorIs(t, err, apperror.ErrInternal)
		})

		t.Run("authnがnilのときErrUnauthenticatedを返しQueryServiceを呼ばない", func(t *testing.T) {
			t.Parallel()

			lt := observability.NewMockUsecaseLayerTracer(t)
			feedQS := mock_query.NewMockPurchaseFeedQueryService(gomock.NewController(t))
			feedQS.EXPECT().FindFeedByUserID(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			feedQS.EXPECT().FindFeedAll(gomock.Any(), gomock.Any()).Times(0)

			u := &usecase{tracer: lt, feedQS: feedQS}
			got, err := u.GetPurchases(context.Background(), nil, ListPurchasesParams{Cursor: firstCursor(t, 2)})
			assert.Nil(t, got)
			require.ErrorIs(t, err, apperror.ErrUnauthenticated)
		})
	})
}

func Test_usecase_findFeedPage(t *testing.T) {
	t.Parallel()

	authnOf := func(t *testing.T, salt string) *auth.Authn {
		t.Helper()
		a, err := auth.New("feed-page-subject", "feed-page-issuer", nil, nil)
		require.NoError(t, err)
		resolved, err := a.WithUserID(uuidtestkit.NewTestFromSalt(t, salt))
		require.NoError(t, err)
		return resolved
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("既定では認可を求めず認証主体の購入だけを読む", func(t *testing.T) {
			t.Parallel()

			lt := observability.NewMockUsecaseLayerTracer(t)
			ctrl := gomock.NewController(t)
			feedQS := mock_query.NewMockPurchaseFeedQueryService(ctrl)
			authorizer := mock_authz.NewMockAuthorizer(ctrl)

			// 自分の購入を読むのに admin の能力は要らない。
			authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			feedQS.EXPECT().FindFeedAll(gomock.Any(), gomock.Any()).Times(0)

			userID := uuidtestkit.NewTestFromSalt(t, "owner")
			var capturedUserID uuid.UUID
			feedQS.EXPECT().FindFeedByUserID(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, uid uuid.UUID, _ query.ListFeedParams) ([]query.PurchaseFeedReadModel, error) {
					capturedUserID = uid
					return []query.PurchaseFeedReadModel{}, nil
				},
			)

			u := &usecase{tracer: lt, feedQS: feedQS, authorizer: authorizer}
			authn, err := auth.New("feed-page-subject", "feed-page-issuer", nil, nil)
			require.NoError(t, err)
			resolved, err := authn.WithUserID(userID)
			require.NoError(t, err)

			got, err := u.findFeedPage(context.Background(), resolved, false, query.ListFeedParams{Limit: 3})
			require.NoError(t, err)
			assert.Empty(t, got)
			assert.Equal(t, userID, capturedUserID)
		})

		t.Run("他ユーザーを含める指定はadmin認可のうえ所有権で閉じない読み取りへ切り替わる", func(t *testing.T) {
			t.Parallel()

			lt := observability.NewMockUsecaseLayerTracer(t)
			ctrl := gomock.NewController(t)
			feedQS := mock_query.NewMockPurchaseFeedQueryService(ctrl)
			authorizer := mock_authz.NewMockAuthorizer(ctrl)

			// 所有権で閉じる読み取りは呼ばれない（母集団が混ざらないことの固定）。
			feedQS.EXPECT().FindFeedByUserID(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			authorizer.EXPECT().Authorize(
				gomock.Any(), gomock.Any(), authz.ActionPurchaseReadAll, gomock.Any(),
			).Return(nil)

			var captured query.ListFeedParams
			feedQS.EXPECT().FindFeedAll(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, params query.ListFeedParams) ([]query.PurchaseFeedReadModel, error) {
					captured = params
					return []query.PurchaseFeedReadModel{}, nil
				},
			)

			u := &usecase{tracer: lt, feedQS: feedQS, authorizer: authorizer}
			got, err := u.findFeedPage(
				context.Background(), authnOf(t, "admin"), true,
				query.ListFeedParams{Limit: 3, StatusCodes: []int16{8}},
			)
			require.NoError(t, err)
			assert.Empty(t, got)
			// 母集団が変わっても件数と絞り込みの条件はそのまま渡る。
			assert.Equal(t, int32(3), captured.Limit)
			assert.Equal(t, []int16{8}, captured.StatusCodes)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("他ユーザーを含める指定が認可されないとき所有権で閉じない読み取りを呼ばない", func(t *testing.T) {
			t.Parallel()

			lt := observability.NewMockUsecaseLayerTracer(t)
			ctrl := gomock.NewController(t)
			feedQS := mock_query.NewMockPurchaseFeedQueryService(ctrl)
			authorizer := mock_authz.NewMockAuthorizer(ctrl)

			feedQS.EXPECT().FindFeedAll(gomock.Any(), gomock.Any()).Times(0)
			// 拒否された指定が自分の購入一覧へ降格しないことも同時に固定する。
			feedQS.EXPECT().FindFeedByUserID(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			authorizer.EXPECT().Authorize(
				gomock.Any(), gomock.Any(), authz.ActionPurchaseReadAll, gomock.Any(),
			).Return(apperror.ErrPermissionDenied)

			u := &usecase{tracer: lt, feedQS: feedQS, authorizer: authorizer}
			got, err := u.findFeedPage(context.Background(), authnOf(t, "user"), true, query.ListFeedParams{Limit: 3})
			assert.Nil(t, got)
			require.ErrorIs(t, err, apperror.ErrPermissionDenied)
		})

		t.Run("認証主体のUserIDが未解決のときQueryServiceを呼ばずエラーを返す", func(t *testing.T) {
			t.Parallel()

			// 認証は済んでいるが内部ユーザーが解決できていない Authn では、所有権の絞り込みに渡す
			// userID が定まらない。ゼロ値で全件を引きに行かないことを固定する。
			lt := observability.NewMockUsecaseLayerTracer(t)
			feedQS := mock_query.NewMockPurchaseFeedQueryService(gomock.NewController(t))
			feedQS.EXPECT().FindFeedByUserID(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			unresolved, err := auth.New("feed-page-subject", "feed-page-issuer", nil, nil)
			require.NoError(t, err)

			u := &usecase{tracer: lt, feedQS: feedQS}
			got, err := u.findFeedPage(context.Background(), unresolved, false, query.ListFeedParams{Limit: 3})
			assert.Nil(t, got)
			require.ErrorIs(t, err, auth.ErrUserIDUnresolved)
		})
	})
}

func Test_encodePurchaseCursor(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("注文日時とIDを符号化したカーソルはナノ秒精度のまま復号できる", func(t *testing.T) {
			t.Parallel()

			orderedAt := time.Date(2099, time.January, 2, 3, 4, 5, 123456789, time.UTC)
			id := uuidtestkit.NewTestFromSalt(t, "encode_cursor_last")

			encoded := encodePurchaseCursor(query.PurchaseFeedReadModel{OrderedAt: orderedAt, ID: id})
			assert.NotEmpty(t, encoded)

			first := 2
			cursor, err := paging.NewCursor(&encoded, &first)
			require.NoError(t, err)

			decoded, err := decodePurchaseCursor(cursor)
			require.NoError(t, err)
			require.NotNil(t, decoded)
			assert.True(t, orderedAt.Equal(decoded.orderedAt))
			assert.Equal(t, id, decoded.id)
		})

		t.Run("keyset境界は末尾行の注文日時とIDのみで決まりステータスなど他項目に影響されない", func(t *testing.T) {
			t.Parallel()

			orderedAt := time.Date(2099, time.January, 2, 3, 4, 5, 0, time.UTC)
			id := uuidtestkit.NewTestFromSalt(t, "encode_cursor_same_key")

			base := encodePurchaseCursor(query.PurchaseFeedReadModel{OrderedAt: orderedAt, ID: id})
			other := encodePurchaseCursor(query.PurchaseFeedReadModel{
				Code:        "code-other",
				TotalAmount: 999,
				StatusID:    uuidtestkit.NewTestFromSalt(t, "encode_cursor_other_status"),
				StatusName:  "キャンセル",
				OrderedAt:   orderedAt,
				ID:          id,
			})

			assert.Equal(t, base, other)
		})
	})
}
