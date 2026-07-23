package purchase

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	domainpurchase "go-boilerplate/internal/domain/purchase"
	mock_purchase "go-boilerplate/internal/domain/purchase/mock"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/exchangerate"
	mock_exchangerate "go-boilerplate/internal/usecase/exchangerate/mock"
	mock_outbox "go-boilerplate/internal/usecase/outbox/mock"
	mock_command "go-boilerplate/internal/usecase/purchase/command/mock"
	"go-boilerplate/internal/usecase/testkit"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// rereadPurchase は、repo.FindByID が返す再構築済みの購入を生成するテストヘルパーです。
func rereadPurchase(t *testing.T) *domainpurchase.Purchase {
	t.Helper()
	details := []domainpurchase.PurchaseDetail{
		domainpurchase.NewPurchaseDetail(
			uuid.NewTestFromSalt(t, "reread_detail"),
			uuid.NewTestFromSalt(t, "reread_product"),
			2, 80000,
		),
	}
	p, err := domainpurchase.Reconstruct(
		uuid.NewTestFromSalt(t, "reread_id"),
		"reread-code",
		uuid.NewTestFromSalt(t, "reread_user"),
		uuid.NewTestFromSalt(t, "reread_status"),
		160000, 16000, 500, 176500,
		details,
		time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC),
	)
	require.NoError(t, err)
	return p
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("依存を注入したUsecaseを生成する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)
			txm := testkit.NewMockTransactionManager(t)
			cmd := mock_command.NewMockCommandService(ctrl)
			repo := mock_purchase.NewMockRepository(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)
			xr := mock_exchangerate.NewMockUsecase(ctrl)

			actual := New(txm, cmd, repo, emit, xr, tf)
			require.NotNil(t, actual)
		})
	})
}

func Test_usecase_CreatePurchase(t *testing.T) {
	t.Parallel()

	productA := uuid.NewTestFromSalt(t, "cp_product")
	// newUsecase は、指定 mock を注入した usecase を生成するローカルヘルパーです。
	newUsecase := func(t *testing.T, cmd *mock_command.MockCommandService, repo *mock_purchase.MockRepository, emit *mock_outbox.MockEmitUsecase, xr *mock_exchangerate.MockUsecase) *usecase {
		t.Helper()
		return &usecase{
			tracer: observability.NewNoopTracerFactory(t).Usecase(),
			txm:    testkit.NewMockTransactionManager(t),
			cmd:    cmd, repo: repo, emit: emit, xr: xr,
		}
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ロック→生成→書き込み→emit→再検証を経て購入ビューを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cmd := mock_command.NewMockCommandService(ctrl)
			repo := mock_purchase.NewMockRepository(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)
			xr := mock_exchangerate.NewMockUsecase(ctrl)

			locked := []domainpurchase.LockedProduct{domainpurchase.NewLockedProduct(productA, 80000, 20)}
			cmd.EXPECT().LockProducts(gomock.Any(), gomock.Any()).Return(locked, nil)
			cmd.EXPECT().CreatePurchase(gomock.Any(), gomock.Any()).Return(nil)
			emit.EXPECT().Emit(gomock.Any(), gomock.Any()).Return(uuid.UUID{}, nil)
			reread := rereadPurchase(t)
			repo.EXPECT().FindByID(gomock.Any(), gomock.Any()).Return(reread, nil)

			u := newUsecase(t, cmd, repo, emit, xr)

			view, err := u.CreatePurchase(context.Background(), CreatePurchaseParams{
				UserID:  uuid.NewTestFromSalt(t, "cp_user"),
				Details: []DetailParam{{ProductID: productA, Quantity: 2}},
			})
			require.NoError(t, err)
			// 再検証で読み直したエンティティが DTO へ写像される。
			assert.Equal(t, reread.ID(), view.ID)
			assert.Equal(t, reread.Code(), view.Code)
			assert.Equal(t, reread.UserID(), view.UserID)
			assert.Equal(t, reread.StatusID(), view.StatusID)
			assert.Equal(t, reread.SubtotalAmount(), view.SubtotalAmount)
			assert.Equal(t, reread.TaxAmount(), view.TaxAmount)
			assert.Equal(t, reread.ShippingFee(), view.ShippingFee)
			assert.Equal(t, reread.TotalAmount(), view.TotalAmount)
			assert.Equal(t, reread.OrderedAt(), view.OrderedAt)
			assert.Nil(t, view.ReferenceAmount)
			require.Len(t, view.Details, 1)
			assert.Equal(t, reread.Details()[0].ProductID(), view.Details[0].ProductID)
			assert.Equal(t, reread.Details()[0].UnitPrice(), view.Details[0].UnitPrice)
		})

		t.Run("displayCurrency指定時は参考換算額を付与する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cmd := mock_command.NewMockCommandService(ctrl)
			repo := mock_purchase.NewMockRepository(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)
			xr := mock_exchangerate.NewMockUsecase(ctrl)

			locked := []domainpurchase.LockedProduct{domainpurchase.NewLockedProduct(productA, 80000, 20)}
			cmd.EXPECT().LockProducts(gomock.Any(), gomock.Any()).Return(locked, nil)
			cmd.EXPECT().CreatePurchase(gomock.Any(), gomock.Any()).Return(nil)
			emit.EXPECT().Emit(gomock.Any(), gomock.Any()).Return(uuid.UUID{}, nil)
			repo.EXPECT().FindByID(gomock.Any(), gomock.Any()).Return(rereadPurchase(t), nil)
			xr.EXPECT().Convert(gomock.Any(), gomock.Any()).Return(&exchangerate.ConvertResult{
				Reference: &exchangerate.ReferenceAmount{Currency: "JPY", Amount: 26475, Rate: 150.5, RateDate: "2026-07-21"},
			}, nil)

			u := newUsecase(t, cmd, repo, emit, xr)

			jpy := "JPY"
			view, err := u.CreatePurchase(context.Background(), CreatePurchaseParams{
				UserID:          uuid.NewTestFromSalt(t, "cp_user2"),
				Details:         []DetailParam{{ProductID: productA, Quantity: 2}},
				DisplayCurrency: &jpy,
			})
			require.NoError(t, err)
			require.NotNil(t, view.ReferenceAmount)
			assert.Equal(t, "JPY", view.ReferenceAmount.Currency)
			assert.Equal(t, int64(26475), view.ReferenceAmount.Amount)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		enoughStock := []domainpurchase.LockedProduct{domainpurchase.NewLockedProduct(productA, 80000, 20)}
		validParams := CreatePurchaseParams{
			UserID:  uuid.NewTestFromSalt(t, "cp_user_err"),
			Details: []DetailParam{{ProductID: productA, Quantity: 2}},
		}

		t.Run("在庫不足の場合、ErrInsufficientStockを伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cmd := mock_command.NewMockCommandService(ctrl)
			// 在庫 1 に対し 2 を要求 → domain New が ErrInsufficientStock
			locked := []domainpurchase.LockedProduct{domainpurchase.NewLockedProduct(productA, 80000, 1)}
			cmd.EXPECT().LockProducts(gomock.Any(), gomock.Any()).Return(locked, nil)

			u := newUsecase(
				t,
				cmd,
				mock_purchase.NewMockRepository(ctrl),
				mock_outbox.NewMockEmitUsecase(ctrl),
				mock_exchangerate.NewMockUsecase(ctrl),
			)

			_, err := u.CreatePurchase(context.Background(), validParams)
			require.ErrorIs(t, err, domainpurchase.ErrInsufficientStock)
		})

		t.Run("LockProductsが失敗した場合はエラーを伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cmd := mock_command.NewMockCommandService(ctrl)
			cmd.EXPECT().LockProducts(gomock.Any(), gomock.Any()).Return(nil, apperror.ErrUnavailable)

			u := newUsecase(
				t,
				cmd,
				mock_purchase.NewMockRepository(ctrl),
				mock_outbox.NewMockEmitUsecase(ctrl),
				mock_exchangerate.NewMockUsecase(ctrl),
			)

			_, err := u.CreatePurchase(context.Background(), validParams)
			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})

		t.Run("CreatePurchase書き込みが失敗した場合はエラーを伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cmd := mock_command.NewMockCommandService(ctrl)
			cmd.EXPECT().LockProducts(gomock.Any(), gomock.Any()).Return(enoughStock, nil)
			cmd.EXPECT().CreatePurchase(gomock.Any(), gomock.Any()).Return(apperror.ErrConflict)

			u := newUsecase(
				t,
				cmd,
				mock_purchase.NewMockRepository(ctrl),
				mock_outbox.NewMockEmitUsecase(ctrl),
				mock_exchangerate.NewMockUsecase(ctrl),
			)

			_, err := u.CreatePurchase(context.Background(), validParams)
			require.ErrorIs(t, err, apperror.ErrConflict)
		})

		t.Run("outbox発行が失敗した場合はエラーを伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cmd := mock_command.NewMockCommandService(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)
			cmd.EXPECT().LockProducts(gomock.Any(), gomock.Any()).Return(enoughStock, nil)
			cmd.EXPECT().CreatePurchase(gomock.Any(), gomock.Any()).Return(nil)
			emit.EXPECT().Emit(gomock.Any(), gomock.Any()).Return(uuid.UUID{}, apperror.ErrInternal)

			u := newUsecase(t, cmd, mock_purchase.NewMockRepository(ctrl), emit, mock_exchangerate.NewMockUsecase(ctrl))

			_, err := u.CreatePurchase(context.Background(), validParams)
			require.ErrorIs(t, err, apperror.ErrInternal)
		})

		t.Run("再検証の読み直しが失敗した場合はエラーを伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cmd := mock_command.NewMockCommandService(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)
			repo := mock_purchase.NewMockRepository(ctrl)
			cmd.EXPECT().LockProducts(gomock.Any(), gomock.Any()).Return(enoughStock, nil)
			cmd.EXPECT().CreatePurchase(gomock.Any(), gomock.Any()).Return(nil)
			emit.EXPECT().Emit(gomock.Any(), gomock.Any()).Return(uuid.UUID{}, nil)
			repo.EXPECT().FindByID(gomock.Any(), gomock.Any()).Return(nil, apperror.ErrNotFound)

			u := newUsecase(t, cmd, repo, emit, mock_exchangerate.NewMockUsecase(ctrl))

			_, err := u.CreatePurchase(context.Background(), validParams)
			require.ErrorIs(t, err, apperror.ErrNotFound)
		})
	})
}

func Test_usecase_referenceAmount(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("換算成功時は参考換算額を返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			xr := mock_exchangerate.NewMockUsecase(ctrl)
			xr.EXPECT().Convert(gomock.Any(), gomock.Any()).Return(&exchangerate.ConvertResult{
				Reference: &exchangerate.ReferenceAmount{Currency: "JPY", Amount: 15050, Rate: 150.5, RateDate: "2026-07-21"},
			}, nil)

			u := &usecase{tracer: observability.NewNoopTracerFactory(t).Usecase(), xr: xr}
			actual := u.referenceAmount(context.Background(), 10000, "JPY")
			require.NotNil(t, actual)
			assert.Equal(t, "JPY", actual.Currency)
			assert.Equal(t, int64(15050), actual.Amount)
			assert.InDelta(t, 150.5, actual.Rate, 0.001)
			assert.Equal(t, "2026-07-21", actual.RateDate)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("換算失敗時はnilでdegradeする", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			xr := mock_exchangerate.NewMockUsecase(ctrl)
			xr.EXPECT().Convert(gomock.Any(), gomock.Any()).Return(nil, xerrors.New("gateway down"))

			u := &usecase{tracer: observability.NewNoopTracerFactory(t).Usecase(), xr: xr}
			assert.Nil(t, u.referenceAmount(context.Background(), 10000, "JPY"))
		})

		t.Run("参考換算額が無い場合はnilを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			xr := mock_exchangerate.NewMockUsecase(ctrl)
			xr.EXPECT().Convert(gomock.Any(), gomock.Any()).Return(&exchangerate.ConvertResult{Reference: nil}, nil)

			u := &usecase{tracer: observability.NewNoopTracerFactory(t).Usecase(), xr: xr}
			assert.Nil(t, u.referenceAmount(context.Background(), 10000, "JPY"))
		})
	})
}

func Test_buildCreatedPayload(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("購入の自己完結スナップショットJSONを生成する", func(t *testing.T) {
			t.Parallel()

			productA := uuid.NewTestFromSalt(t, "bp_product")
			entity, err := domainpurchase.New(
				uuid.NewTestFromSalt(t, "bp_id"),
				"bp-code",
				uuid.NewTestFromSalt(t, "bp_user"),
				[]domainpurchase.DetailInput{{ID: uuid.NewTestFromSalt(t, "bp_d"), ProductID: productA, Quantity: 2}},
				[]domainpurchase.LockedProduct{domainpurchase.NewLockedProduct(productA, 80000, 20)},
			)
			require.NoError(t, err)

			payload, perr := buildCreatedPayload(entity)
			require.NoError(t, perr)

			var decoded struct {
				PurchaseID     string `json:"purchaseId"`
				Code           string `json:"code"`
				UserID         string `json:"userId"`
				StatusCode     int    `json:"statusCode"`
				SubtotalAmount int    `json:"subtotalAmount"`
				TaxAmount      int    `json:"taxAmount"`
				ShippingFee    int    `json:"shippingFee"`
				TotalAmount    int    `json:"totalAmount"`
				Details        []struct {
					ProductID string `json:"productId"`
					Quantity  int    `json:"quantity"`
					UnitPrice int    `json:"unitPrice"`
				} `json:"details"`
			}
			require.NoError(t, json.Unmarshal(payload, &decoded))
			assert.Equal(t, entity.ID().String(), decoded.PurchaseID)
			assert.Equal(t, "bp-code", decoded.Code)
			assert.Equal(t, entity.UserID().String(), decoded.UserID)
			assert.Equal(t, domainpurchase.StatusCodeUnprocessed, decoded.StatusCode)
			// subtotal=160000 / tax=16000（切り捨て10%）/ shipping=500 / total=176500
			assert.Equal(t, 160000, decoded.SubtotalAmount)
			assert.Equal(t, 16000, decoded.TaxAmount)
			assert.Equal(t, 500, decoded.ShippingFee)
			assert.Equal(t, 176500, decoded.TotalAmount)
			require.Len(t, decoded.Details, 1)
			assert.Equal(t, productA.String(), decoded.Details[0].ProductID)
			assert.Equal(t, 2, decoded.Details[0].Quantity)
			assert.Equal(t, 80000, decoded.Details[0].UnitPrice)
		})
	})
}

func Test_toPurchaseView(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("購入集約を出力DTOへ変換する", func(t *testing.T) {
			t.Parallel()

			entity := rereadPurchase(t)
			view := toPurchaseView(entity)
			assert.Equal(t, entity.ID(), view.ID)
			assert.Equal(t, entity.Code(), view.Code)
			assert.Equal(t, entity.UserID(), view.UserID)
			assert.Equal(t, entity.StatusID(), view.StatusID)
			assert.Equal(t, entity.SubtotalAmount(), view.SubtotalAmount)
			assert.Equal(t, entity.TaxAmount(), view.TaxAmount)
			assert.Equal(t, entity.ShippingFee(), view.ShippingFee)
			assert.Equal(t, entity.TotalAmount(), view.TotalAmount)
			assert.Equal(t, entity.OrderedAt(), view.OrderedAt)
			require.Len(t, view.Details, 1)
			assert.Equal(t, entity.Details()[0].ProductID(), view.Details[0].ProductID)
			assert.Equal(t, entity.Details()[0].Quantity(), view.Details[0].Quantity)
			assert.Equal(t, entity.Details()[0].UnitPrice(), view.Details[0].UnitPrice)
			assert.Nil(t, view.ReferenceAmount)
		})
	})
}
