package summary

import (
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/observability"
	authbd "go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/internal/usecase/purchase/query"
	mock_query "go-boilerplate/internal/usecase/purchase/query/mock"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func newAuthn(t *testing.T, userID uuid.UUID) *authbd.Authn {
	t.Helper()
	a, err := authbd.New("sub-"+userID.String(), authbd.IssuerMock, nil, nil)
	require.NoError(t, err)

	resolved, err := a.WithUserID(userID)
	require.NoError(t, err)
	return resolved
}

func Test_usecase_GetPurchaseSummary(t *testing.T) {
	t.Parallel()

	newUsecase := func(t *testing.T, qs query.PurchaseSummaryQueryService) *usecase {
		t.Helper()
		return &usecase{
			tracer: observability.NewNoopTracerFactory(t).Usecase(),
			qs:     qs,
		}
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証主体のuserIDをQSへ渡し集計を総計へ畳み込む", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			userID := uuid.NewTestFromSalt(t, "sm_user")
			unprocessedID := uuid.NewTestFromSalt(t, "sm_unprocessed")
			canceledID := uuid.NewTestFromSalt(t, "sm_canceled")

			qs := mock_query.NewMockPurchaseSummaryQueryService(ctrl)
			qs.EXPECT().SummarizeByUserID(gomock.Any(), userID).Return([]query.PurchaseStatusSummaryReadModel{
				{StatusID: unprocessedID, StatusName: "未処理", Count: 2, TotalAmount: 300},
				{StatusID: canceledID, StatusName: "キャンセル", Count: 1, TotalAmount: 150},
			}, nil)

			actual, err := newUsecase(t, qs).GetPurchaseSummary(t.Context(), newAuthn(t, userID))
			require.NoError(t, err)

			assert.Equal(t, int64(3), actual.TotalCount)
			assert.Equal(t, int64(450), actual.TotalAmount)
			assert.Equal(t, []StatusCountView{
				{StatusID: unprocessedID, StatusName: "未処理", Count: 2, TotalAmount: 300},
				{StatusID: canceledID, StatusName: "キャンセル", Count: 1, TotalAmount: 150},
			}, actual.StatusBreakdown)
		})

		t.Run("購入が1件もない場合はnilではない空の内訳とゼロ値を返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			userID := uuid.NewTestFromSalt(t, "sm_empty_user")

			qs := mock_query.NewMockPurchaseSummaryQueryService(ctrl)
			qs.EXPECT().SummarizeByUserID(gomock.Any(), userID).Return([]query.PurchaseStatusSummaryReadModel{}, nil)

			actual, err := newUsecase(t, qs).GetPurchaseSummary(t.Context(), newAuthn(t, userID))
			require.NoError(t, err)
			assert.Equal(t, int64(0), actual.TotalCount)
			assert.Equal(t, int64(0), actual.TotalAmount)
			assert.NotNil(t, actual.StatusBreakdown)
			assert.Empty(t, actual.StatusBreakdown)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証コンテキストがnilの場合、ErrUnauthenticatedを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			qs := mock_query.NewMockPurchaseSummaryQueryService(ctrl)

			actual, err := newUsecase(t, qs).GetPurchaseSummary(t.Context(), nil)
			require.ErrorIs(t, err, apperror.ErrUnauthenticated)
			assert.Equal(t, SummaryView{}, actual)
		})

		t.Run("内部UserIDが未解決の場合、エラーを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			qs := mock_query.NewMockPurchaseSummaryQueryService(ctrl)

			// WithUserID を経ていない Authn は内部 UserID を解決できない。
			authn, err := authbd.New("sub-unresolved", authbd.IssuerMock, nil, nil)
			require.NoError(t, err)

			actual, err := newUsecase(t, qs).GetPurchaseSummary(t.Context(), authn)
			require.ErrorIs(t, err, authbd.ErrUserIDUnresolved)
			assert.Equal(t, SummaryView{}, actual)
		})

		t.Run("QSがエラーを返した場合、そのまま伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			userID := uuid.NewTestFromSalt(t, "sm_err_user")
			expected := xerrors.Wrap(apperror.ErrInternal, "query service failed")

			qs := mock_query.NewMockPurchaseSummaryQueryService(ctrl)
			qs.EXPECT().SummarizeByUserID(gomock.Any(), userID).Return(nil, expected)

			actual, err := newUsecase(t, qs).GetPurchaseSummary(t.Context(), newAuthn(t, userID))
			require.ErrorIs(t, err, expected)
			assert.Equal(t, SummaryView{}, actual)
		})
	})
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("依存を注入したユースケース実装を生成する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			qs := mock_query.NewMockPurchaseSummaryQueryService(ctrl)
			tf := observability.NewNoopTracerFactory(t)

			actual, ok := New(qs, tf).(*usecase)
			require.True(t, ok)
			assert.Equal(t, qs, actual.qs)
			assert.NotNil(t, actual.tracer)
		})
	})
}

func Test_toSummaryView(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ステータス別の件数と金額を総計へ畳み込み内訳の順序を保持する", func(t *testing.T) {
			t.Parallel()

			first := uuid.NewTestFromSalt(t, "tv_first")
			second := uuid.NewTestFromSalt(t, "tv_second")

			actual := toSummaryView([]query.PurchaseStatusSummaryReadModel{
				{StatusID: first, StatusName: "未処理", Count: 1, TotalAmount: 100},
				{StatusID: second, StatusName: "完了", Count: 3, TotalAmount: 900},
			})

			assert.Equal(t, int64(4), actual.TotalCount)
			assert.Equal(t, int64(1000), actual.TotalAmount)
			assert.Equal(t, []StatusCountView{
				{StatusID: first, StatusName: "未処理", Count: 1, TotalAmount: 100},
				{StatusID: second, StatusName: "完了", Count: 3, TotalAmount: 900},
			}, actual.StatusBreakdown)
		})

		t.Run("集計結果が空の場合はnilではない空の内訳とゼロ値を返す", func(t *testing.T) {
			t.Parallel()

			actual := toSummaryView(nil)

			assert.Equal(t, int64(0), actual.TotalCount)
			assert.Equal(t, int64(0), actual.TotalAmount)
			assert.NotNil(t, actual.StatusBreakdown)
			assert.Empty(t, actual.StatusBreakdown)
		})
	})
}
