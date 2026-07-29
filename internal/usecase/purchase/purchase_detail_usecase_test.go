package purchase

import (
	"context"
	"testing"
	"time"

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

func Test_usecase_GetPurchaseDetail(t *testing.T) {
	t.Parallel()

	newAuthn := func(t *testing.T, userID uuid.UUID) *authbd.Authn {
		t.Helper()
		a, err := authbd.New("sub-"+userID.String(), authbd.IssuerMock, nil, nil)
		require.NoError(t, err)
		return a.WithUserID(userID)
	}

	newUsecase := func(t *testing.T, qs query.PurchaseDetailQueryService) *usecase {
		t.Helper()
		return &usecase{
			tracer:   observability.NewNoopTracerFactory(t).Usecase(),
			detailQS: qs,
		}
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証主体のuserIDをQSへ渡し読み取りモデルをビューへ写像する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			userID := uuid.NewTestFromSalt(t, "gd_user")
			purchaseID := uuid.NewTestFromSalt(t, "gd_purchase")
			paidAt := time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)

			rm := &query.PurchaseDetailReadModel{
				ID:             purchaseID,
				Code:           "gd-code",
				UserID:         userID,
				StatusID:       uuid.NewTestFromSalt(t, "gd_status"),
				StatusName:     "支払い済み",
				SubtotalAmount: 160000,
				TaxAmount:      16000,
				ShippingFee:    500,
				TotalAmount:    176500,
				Items: []query.PurchaseDetailItem{
					{ProductID: uuid.NewTestFromSalt(t, "gd_product"), ProductName: "商品A", Quantity: 2, UnitPrice: mustPrice(t, "800")},
				},
				OrderedAt:  time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC),
				PaidAt:     &paidAt,
				CanceledAt: nil,
			}

			var capturedUserID, capturedPurchaseID uuid.UUID
			qs := mock_query.NewMockPurchaseDetailQueryService(ctrl)
			qs.EXPECT().FindDetailByUserAndID(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, uid, pid uuid.UUID) (*query.PurchaseDetailReadModel, error) {
					capturedUserID = uid
					capturedPurchaseID = pid
					return rm, nil
				},
			)

			u := newUsecase(t, qs)
			view, err := u.GetPurchaseDetail(context.Background(), newAuthn(t, userID), purchaseID)
			require.NoError(t, err)

			assert.Equal(t, userID, capturedUserID)
			assert.Equal(t, purchaseID, capturedPurchaseID)
			assert.Equal(t, rm.Code, view.Code)
			assert.Equal(t, rm.StatusID, view.StatusID)
			assert.Equal(t, rm.StatusName, view.StatusName)
			assert.Equal(t, int64(16000), view.TaxAmount)
			assert.Equal(t, int64(500), view.ShippingFee)
			assert.Equal(t, int64(176500), view.TotalAmount)
			assert.Equal(t, rm.PaidAt, view.PaidAt)
			assert.Nil(t, view.CanceledAt)
			require.Len(t, view.Details, 1)
			assert.Equal(t, "商品A", view.Details[0].ProductName)
			assert.Equal(t, "800", view.Details[0].UnitPrice.String())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("authnがnilの場合はUnauthenticatedを返しQSは呼ばれない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			qs := mock_query.NewMockPurchaseDetailQueryService(ctrl)
			qs.EXPECT().FindDetailByUserAndID(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			u := newUsecase(t, qs)
			_, err := u.GetPurchaseDetail(context.Background(), nil, uuid.NewTestFromSalt(t, "gd_nil"))
			require.ErrorIs(t, err, apperror.ErrUnauthenticated)
		})

		t.Run("authnはあるがUserIDが未解決の場合はErrUserIDUnresolvedを伝播しQSは呼ばれない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			qs := mock_query.NewMockPurchaseDetailQueryService(ctrl)
			qs.EXPECT().FindDetailByUserAndID(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			// WithUserID を呼ばず内部 UserID を未解決のままにした authn を渡す。
			unresolved, err := authbd.New("sub-unresolved", authbd.IssuerMock, nil, nil)
			require.NoError(t, err)

			u := newUsecase(t, qs)
			_, err = u.GetPurchaseDetail(context.Background(), unresolved, uuid.NewTestFromSalt(t, "gd_unresolved"))
			require.ErrorIs(t, err, authbd.ErrUserIDUnresolved)
		})

		t.Run("QSのNotFoundをそのまま伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			userID := uuid.NewTestFromSalt(t, "gd_nf_user")
			qs := mock_query.NewMockPurchaseDetailQueryService(ctrl)
			qs.EXPECT().FindDetailByUserAndID(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, xerrors.Wrap(apperror.ErrNotFound, "not found"))

			u := newUsecase(t, qs)
			_, err := u.GetPurchaseDetail(context.Background(), newAuthn(t, userID), uuid.NewTestFromSalt(t, "gd_nf"))
			require.ErrorIs(t, err, apperror.ErrNotFound)
		})
	})
}

func Test_toPurchaseGetDetailView(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("読み取りモデルを単価decimal込みのビューへ写像する", func(t *testing.T) {
			t.Parallel()

			paidAt := time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)
			rm := &query.PurchaseDetailReadModel{
				ID:             uuid.NewTestFromSalt(t, "tv_id"),
				Code:           "tv-code",
				UserID:         uuid.NewTestFromSalt(t, "tv_user"),
				StatusID:       uuid.NewTestFromSalt(t, "tv_status"),
				StatusName:     "支払い済み",
				SubtotalAmount: 160000,
				TaxAmount:      16000,
				ShippingFee:    500,
				TotalAmount:    176500,
				Items: []query.PurchaseDetailItem{
					{ProductID: uuid.NewTestFromSalt(t, "tv_prod"), ProductName: "商品A", Quantity: 2, UnitPrice: mustPrice(t, "800")},
				},
				OrderedAt:  time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC),
				PaidAt:     &paidAt,
				CanceledAt: nil,
			}

			view := toPurchaseGetDetailView(rm)
			assert.Equal(t, rm.ID, view.ID)
			assert.Equal(t, rm.Code, view.Code)
			assert.Equal(t, rm.StatusName, view.StatusName)
			assert.Equal(t, int64(16000), view.TaxAmount)
			assert.Equal(t, int64(500), view.ShippingFee)
			assert.Equal(t, int64(176500), view.TotalAmount)
			assert.Equal(t, rm.PaidAt, view.PaidAt)
			assert.Nil(t, view.CanceledAt)
			require.Len(t, view.Details, 1)
			assert.Equal(t, "商品A", view.Details[0].ProductName)
			assert.Equal(t, "800", view.Details[0].UnitPrice.String())
		})

		t.Run("キャンセル済み読み取りモデルはcanceledAtを写像しpaidAtはnilになる", func(t *testing.T) {
			t.Parallel()

			canceledAt := time.Date(2026, time.July, 26, 9, 0, 0, 0, time.UTC)
			rm := &query.PurchaseDetailReadModel{
				ID:             uuid.NewTestFromSalt(t, "tvc_id"),
				Code:           "tvc-code",
				UserID:         uuid.NewTestFromSalt(t, "tvc_user"),
				StatusID:       uuid.NewTestFromSalt(t, "tvc_status"),
				StatusName:     "キャンセル",
				SubtotalAmount: 160000,
				TaxAmount:      16000,
				ShippingFee:    500,
				TotalAmount:    176500,
				Items: []query.PurchaseDetailItem{
					{ProductID: uuid.NewTestFromSalt(t, "tvc_prod"), ProductName: "商品C", Quantity: 1, UnitPrice: mustPrice(t, "800")},
				},
				OrderedAt:  time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC),
				PaidAt:     nil,
				CanceledAt: &canceledAt,
			}

			view := toPurchaseGetDetailView(rm)
			assert.Nil(t, view.PaidAt)
			require.NotNil(t, view.CanceledAt)
			assert.Equal(t, rm.CanceledAt, view.CanceledAt)
		})
	})
}
