package purchase

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/lexicon/money"
	domainpurchase "go-boilerplate/internal/domain/purchase"
	mock_purchase "go-boilerplate/internal/domain/purchase/mock"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/internal/usecase/boundary/authz"
	mock_authz "go-boilerplate/internal/usecase/boundary/authz/mock"
	clocktestkit "go-boilerplate/internal/usecase/boundary/clock/testkit"
	mock_tx "go-boilerplate/internal/usecase/boundary/tx/mock"
	"go-boilerplate/internal/usecase/outbox"
	mock_outbox "go-boilerplate/internal/usecase/outbox/mock"
	"go-boilerplate/internal/usecase/purchase/event"
	mock_query "go-boilerplate/internal/usecase/purchase/query/mock"
	"go-boilerplate/internal/usecase/testkit"
	decimaltestkit "go-boilerplate/pkg/decimal/testkit"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// mustPrice は、テスト用に十進文字列（ドル）から非負の money.Price を構築します。
//
//nolint:unparam // テスト補助ヘルパー。現行の呼び出しは同一値だが用途は可変
func mustPrice(t *testing.T, s string) money.Price {
	t.Helper()
	p, err := money.NewPrice(decimaltestkit.MustParse(t, s))
	require.NoError(t, err)
	return p
}

// rereadPurchase は、repo.FindByID が返す再構築済みの購入を生成するテストヘルパーです。
func rereadPurchase(t *testing.T) *domainpurchase.Purchase {
	t.Helper()
	details := []domainpurchase.PurchaseDetail{
		domainpurchase.NewPurchaseDetail(uuidtestkit.NewTestFromSalt(t, "reread_detail"), domainpurchase.PurchaseDetailAttributes{
			ProductID: uuidtestkit.NewTestFromSalt(t, "reread_product"),
			Quantity:  2,
			UnitPrice: mustPrice(t, "800"),
		}),
	}
	p, err := domainpurchase.Reconstruct(uuidtestkit.NewTestFromSalt(t, "reread_id"), domainpurchase.Attributes{
		Code:           "reread-code",
		UserID:         uuidtestkit.NewTestFromSalt(t, "reread_user"),
		StatusID:       uuidtestkit.NewTestFromSalt(t, "reread_status"),
		StatusCode:     domainpurchase.StatusUnprocessed.Code(),
		SubtotalAmount: 160000,
		TaxAmount:      16000,
		ShippingFee:    500,
		TotalAmount:    176500,
		Details:        details,
		OrderedAt:      time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC),
		PaidAt:         nil,
		CanceledAt:     nil,
		ShippedAt:      nil,
		DeliveredAt:    nil,
	})
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
			cmd := mock_purchase.NewMockCommandService(ctrl)
			repo := mock_purchase.NewMockRepository(ctrl)
			detailQS := mock_query.NewMockPurchaseDetailQueryService(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)
			authorizer := mock_authz.NewMockAuthorizer(ctrl)
			clk := clocktestkit.NewMockClock(t, time.Date(2026, time.July, 25, 0, 0, 0, 0, time.UTC))

			expected := &usecase{
				tracer:     tf.Usecase(),
				txm:        txm,
				cmd:        cmd,
				repo:       repo,
				detailQS:   detailQS,
				emit:       emit,
				clock:      clk,
				authorizer: authorizer,
			}
			actual := New(txm, cmd, repo, detailQS, emit, clk, authorizer, tf)
			assert.Equal(t, expected, actual)
		})
	})
}

func Test_usecase_CreatePurchase(t *testing.T) {
	t.Parallel()

	productA := uuidtestkit.NewTestFromSalt(t, "cp_product")
	// newUsecase は、指定 mock を注入した usecase を生成するローカルヘルパーです。
	newUsecase := func(t *testing.T, cmd *mock_purchase.MockCommandService, repo *mock_purchase.MockRepository, emit *mock_outbox.MockEmitUsecase) *usecase {
		t.Helper()
		return &usecase{
			tracer: observability.NewNoopTracerFactory(t).Usecase(),
			txm:    testkit.NewMockTransactionManager(t),
			cmd:    cmd, repo: repo, emit: emit,
			clock: clocktestkit.NewMockClock(t, time.Date(2026, time.July, 25, 0, 0, 0, 0, time.UTC)),
		}
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ロック→生成→書き込み→emit→再検証を経て購入ビューを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cmd := mock_purchase.NewMockCommandService(ctrl)
			repo := mock_purchase.NewMockRepository(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)

			locked := []domainpurchase.LockedProduct{domainpurchase.NewLockedProduct(productA, mustPrice(t, "800"), 20)}
			cmd.EXPECT().LockProducts(gomock.Any(), gomock.Any()).Return(locked, nil)
			cmd.EXPECT().CreatePurchase(gomock.Any(), gomock.Any()).Return(nil)
			emit.EXPECT().Emit(gomock.Any(), gomock.Any()).Return(uuid.UUID{}, nil)
			reread := rereadPurchase(t)
			repo.EXPECT().FindByID(gomock.Any(), gomock.Any()).Return(reread, nil)

			u := newUsecase(t, cmd, repo, emit)

			view, err := u.CreatePurchase(context.Background(), CreatePurchaseParams{
				UserID:  uuidtestkit.NewTestFromSalt(t, "cp_user"),
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
			require.Len(t, view.Details, 1)
			assert.Equal(t, reread.Details()[0].ProductID(), view.Details[0].ProductID)
			assert.True(t, reread.Details()[0].UnitPrice().Decimal().Equal(view.Details[0].UnitPrice))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		enoughStock := []domainpurchase.LockedProduct{domainpurchase.NewLockedProduct(productA, mustPrice(t, "800"), 20)}
		validParams := CreatePurchaseParams{
			UserID:  uuidtestkit.NewTestFromSalt(t, "cp_user_err"),
			Details: []DetailParam{{ProductID: productA, Quantity: 2}},
		}

		t.Run("在庫不足の場合、ErrInsufficientStockを伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cmd := mock_purchase.NewMockCommandService(ctrl)
			// 在庫 1 に対し 2 を要求 → domain New が ErrInsufficientStock
			locked := []domainpurchase.LockedProduct{domainpurchase.NewLockedProduct(productA, mustPrice(t, "800"), 1)}
			cmd.EXPECT().LockProducts(gomock.Any(), gomock.Any()).Return(locked, nil)

			u := newUsecase(
				t,
				cmd,
				mock_purchase.NewMockRepository(ctrl),
				mock_outbox.NewMockEmitUsecase(ctrl),
			)

			_, err := u.CreatePurchase(context.Background(), validParams)
			require.ErrorIs(t, err, domainpurchase.ErrInsufficientStock)
		})

		t.Run("LockProductsが失敗した場合はエラーを伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cmd := mock_purchase.NewMockCommandService(ctrl)
			cmd.EXPECT().LockProducts(gomock.Any(), gomock.Any()).Return(nil, apperror.ErrUnavailable)

			u := newUsecase(
				t,
				cmd,
				mock_purchase.NewMockRepository(ctrl),
				mock_outbox.NewMockEmitUsecase(ctrl),
			)

			_, err := u.CreatePurchase(context.Background(), validParams)
			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})

		t.Run("CreatePurchase書き込みが失敗した場合はエラーを伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cmd := mock_purchase.NewMockCommandService(ctrl)
			cmd.EXPECT().LockProducts(gomock.Any(), gomock.Any()).Return(enoughStock, nil)
			cmd.EXPECT().CreatePurchase(gomock.Any(), gomock.Any()).Return(apperror.ErrConflict)

			u := newUsecase(
				t,
				cmd,
				mock_purchase.NewMockRepository(ctrl),
				mock_outbox.NewMockEmitUsecase(ctrl),
			)

			_, err := u.CreatePurchase(context.Background(), validParams)
			require.ErrorIs(t, err, apperror.ErrConflict)
		})

		t.Run("outbox発行が失敗した場合はエラーを伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cmd := mock_purchase.NewMockCommandService(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)
			cmd.EXPECT().LockProducts(gomock.Any(), gomock.Any()).Return(enoughStock, nil)
			cmd.EXPECT().CreatePurchase(gomock.Any(), gomock.Any()).Return(nil)
			emit.EXPECT().Emit(gomock.Any(), gomock.Any()).Return(uuid.UUID{}, apperror.ErrInternal)

			u := newUsecase(t, cmd, mock_purchase.NewMockRepository(ctrl), emit)

			_, err := u.CreatePurchase(context.Background(), validParams)
			require.ErrorIs(t, err, apperror.ErrInternal)
		})

		t.Run("再検証の読み直しが失敗した場合はエラーを伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cmd := mock_purchase.NewMockCommandService(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)
			repo := mock_purchase.NewMockRepository(ctrl)
			cmd.EXPECT().LockProducts(gomock.Any(), gomock.Any()).Return(enoughStock, nil)
			cmd.EXPECT().CreatePurchase(gomock.Any(), gomock.Any()).Return(nil)
			emit.EXPECT().Emit(gomock.Any(), gomock.Any()).Return(uuid.UUID{}, nil)
			repo.EXPECT().FindByID(gomock.Any(), gomock.Any()).Return(nil, apperror.ErrNotFound)

			u := newUsecase(t, cmd, repo, emit)

			_, err := u.CreatePurchase(context.Background(), validParams)
			require.ErrorIs(t, err, apperror.ErrNotFound)
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
			assert.True(t, entity.Details()[0].UnitPrice().Decimal().Equal(view.Details[0].UnitPrice))
		})
	})
}

func Test_usecase_CancelPurchase(t *testing.T) {
	t.Parallel()

	userID := uuidtestkit.NewTestFromSalt(t, "cancel_uc_user")
	purchaseID := uuidtestkit.NewTestFromSalt(t, "cancel_uc_id")

	// lockable は、cmd.LockPurchase が返す再構築済み購入を生成するローカルヘルパーです。
	lockable := func(t *testing.T, owner uuid.UUID, status domainpurchase.Status) *domainpurchase.Purchase {
		t.Helper()
		details := []domainpurchase.PurchaseDetail{
			domainpurchase.NewPurchaseDetail(uuidtestkit.NewTestFromSalt(t, "cancel_uc_d1"), domainpurchase.PurchaseDetailAttributes{
				ProductID: uuidtestkit.NewTestFromSalt(t, "cancel_uc_p1"),
				Quantity:  2,
				UnitPrice: mustPrice(t, "800"),
			}),
		}
		p, err := domainpurchase.Reconstruct(purchaseID, domainpurchase.Attributes{
			Code:           "cancel-uc-code",
			UserID:         owner,
			StatusID:       uuidtestkit.NewTestFromSalt(t, "cancel_uc_status"),
			StatusCode:     status.Code(),
			SubtotalAmount: 160000,
			TaxAmount:      16000,
			ShippingFee:    500,
			TotalAmount:    176500,
			Details:        details,
			OrderedAt:      time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC),
			PaidAt:         nil,
			CanceledAt:     nil,
			ShippedAt:      nil,
			DeliveredAt:    nil,
		})
		require.NoError(t, err)
		return p
	}

	// newUC は、指定 mock を注入した usecase を生成するローカルヘルパーです。
	newUC := func(t *testing.T, cmd *mock_purchase.MockCommandService, repo *mock_purchase.MockRepository, emit *mock_outbox.MockEmitUsecase) *usecase {
		t.Helper()
		return &usecase{
			tracer: observability.NewNoopTracerFactory(t).Usecase(),
			txm:    testkit.NewMockTransactionManager(t),
			cmd:    cmd, repo: repo, emit: emit,
			clock: clocktestkit.NewMockClock(t, time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)),
		}
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ロック→所有権検証→キャンセル→在庫復元→emit→再読込を経てキャンセルビューを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cmd := mock_purchase.NewMockCommandService(ctrl)
			repo := mock_purchase.NewMockRepository(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)

			cmd.EXPECT().LockPurchase(gomock.Any(), purchaseID).Return(lockable(t, userID, domainpurchase.StatusUnprocessed), nil)
			cmd.EXPECT().CancelPurchase(gomock.Any(), gomock.Any()).Return(nil)
			emit.EXPECT().Emit(gomock.Any(), gomock.Any()).Return(uuid.UUID{}, nil)

			canceledAt := time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)
			detail := &domainpurchase.Detail{
				ID:             purchaseID,
				Code:           "cancel-uc-code",
				UserID:         userID,
				StatusID:       uuidtestkit.NewTestFromSalt(t, "cancel_uc_canceled_status"),
				StatusName:     "キャンセル",
				SubtotalAmount: 160000,
				TaxAmount:      16000,
				ShippingFee:    500,
				TotalAmount:    176500,
				Details: []domainpurchase.PurchaseDetail{
					domainpurchase.NewPurchaseDetail(uuidtestkit.NewTestFromSalt(t, "cancel_uc_d1"), domainpurchase.PurchaseDetailAttributes{
						ProductID: uuidtestkit.NewTestFromSalt(t, "cancel_uc_p1"),
						Quantity:  2,
						UnitPrice: mustPrice(t, "800"),
					}),
				},
				OrderedAt:  time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC),
				CanceledAt: &canceledAt,
			}
			repo.EXPECT().FindDetailByID(gomock.Any(), purchaseID).Return(detail, nil)

			u := newUC(t, cmd, repo, emit)
			view, err := u.CancelPurchase(context.Background(), CancelPurchaseParams{PurchaseID: purchaseID, UserID: userID})
			require.NoError(t, err)
			assert.Equal(t, detail.StatusID, view.StatusID)
			assert.Equal(t, "キャンセル", view.StatusName)
			require.NotNil(t, view.CanceledAt)
			assert.Equal(t, canceledAt, *view.CanceledAt)
			require.Len(t, view.Details, 1)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("他人の購入の場合、ErrNotFoundを返し在庫復元を行わない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cmd := mock_purchase.NewMockCommandService(ctrl)
			repo := mock_purchase.NewMockRepository(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)

			otherOwner := uuidtestkit.NewTestFromSalt(t, "cancel_uc_other")
			cmd.EXPECT().LockPurchase(gomock.Any(), purchaseID).Return(lockable(t, otherOwner, domainpurchase.StatusUnprocessed), nil)

			u := newUC(t, cmd, repo, emit)
			_, err := u.CancelPurchase(context.Background(), CancelPurchaseParams{PurchaseID: purchaseID, UserID: userID})
			require.ErrorIs(t, err, apperror.ErrNotFound)
		})

		t.Run("不正遷移（完了）の場合、ErrCancelNotAllowedを伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cmd := mock_purchase.NewMockCommandService(ctrl)
			repo := mock_purchase.NewMockRepository(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)

			cmd.EXPECT().LockPurchase(gomock.Any(), purchaseID).Return(lockable(t, userID, domainpurchase.StatusCompleted), nil)

			u := newUC(t, cmd, repo, emit)
			_, err := u.CancelPurchase(context.Background(), CancelPurchaseParams{PurchaseID: purchaseID, UserID: userID})
			require.ErrorIs(t, err, domainpurchase.ErrCancelNotAllowed)
		})

		t.Run("LockPurchaseが失敗した場合はエラーを伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cmd := mock_purchase.NewMockCommandService(ctrl)
			repo := mock_purchase.NewMockRepository(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)

			cmd.EXPECT().LockPurchase(gomock.Any(), purchaseID).Return(nil, apperror.ErrUnavailable)

			u := newUC(t, cmd, repo, emit)
			_, err := u.CancelPurchase(context.Background(), CancelPurchaseParams{PurchaseID: purchaseID, UserID: userID})
			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})

		t.Run("CancelPurchase書き込みが失敗した場合はエラーを伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cmd := mock_purchase.NewMockCommandService(ctrl)
			repo := mock_purchase.NewMockRepository(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)

			cmd.EXPECT().LockPurchase(gomock.Any(), purchaseID).Return(lockable(t, userID, domainpurchase.StatusUnprocessed), nil)
			cmd.EXPECT().CancelPurchase(gomock.Any(), gomock.Any()).Return(apperror.ErrConflict)

			u := newUC(t, cmd, repo, emit)
			_, err := u.CancelPurchase(context.Background(), CancelPurchaseParams{PurchaseID: purchaseID, UserID: userID})
			require.ErrorIs(t, err, apperror.ErrConflict)
		})

		t.Run("outbox発行が失敗した場合はエラーを伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cmd := mock_purchase.NewMockCommandService(ctrl)
			repo := mock_purchase.NewMockRepository(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)

			cmd.EXPECT().LockPurchase(gomock.Any(), purchaseID).Return(lockable(t, userID, domainpurchase.StatusUnprocessed), nil)
			cmd.EXPECT().CancelPurchase(gomock.Any(), gomock.Any()).Return(nil)
			emit.EXPECT().Emit(gomock.Any(), gomock.Any()).Return(uuid.UUID{}, apperror.ErrInternal)

			u := newUC(t, cmd, repo, emit)
			_, err := u.CancelPurchase(context.Background(), CancelPurchaseParams{PurchaseID: purchaseID, UserID: userID})
			require.ErrorIs(t, err, apperror.ErrInternal)
		})

		t.Run("再読込が失敗した場合はエラーを伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cmd := mock_purchase.NewMockCommandService(ctrl)
			repo := mock_purchase.NewMockRepository(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)

			cmd.EXPECT().LockPurchase(gomock.Any(), purchaseID).Return(lockable(t, userID, domainpurchase.StatusUnprocessed), nil)
			cmd.EXPECT().CancelPurchase(gomock.Any(), gomock.Any()).Return(nil)
			emit.EXPECT().Emit(gomock.Any(), gomock.Any()).Return(uuid.UUID{}, nil)
			repo.EXPECT().FindDetailByID(gomock.Any(), purchaseID).Return(nil, apperror.ErrNotFound)

			u := newUC(t, cmd, repo, emit)
			_, err := u.CancelPurchase(context.Background(), CancelPurchaseParams{PurchaseID: purchaseID, UserID: userID})
			require.ErrorIs(t, err, apperror.ErrNotFound)
		})
	})
}

func Test_usecase_PayPurchase(t *testing.T) {
	t.Parallel()

	userID := uuidtestkit.NewTestFromSalt(t, "pay_uc_user")
	purchaseID := uuidtestkit.NewTestFromSalt(t, "pay_uc_id")

	// lockable は、cmd.LockPurchase が返す再構築済み購入を生成するローカルヘルパーです。
	lockable := func(t *testing.T, owner uuid.UUID, status domainpurchase.Status, paidAt *time.Time) *domainpurchase.Purchase {
		t.Helper()
		details := []domainpurchase.PurchaseDetail{
			domainpurchase.NewPurchaseDetail(uuidtestkit.NewTestFromSalt(t, "pay_uc_d1"), domainpurchase.PurchaseDetailAttributes{
				ProductID: uuidtestkit.NewTestFromSalt(t, "pay_uc_p1"),
				Quantity:  2,
				UnitPrice: mustPrice(t, "800"),
			}),
		}
		p, err := domainpurchase.Reconstruct(purchaseID, domainpurchase.Attributes{
			Code:           "pay-uc-code",
			UserID:         owner,
			StatusID:       uuidtestkit.NewTestFromSalt(t, "pay_uc_status"),
			StatusCode:     status.Code(),
			SubtotalAmount: 160000,
			TaxAmount:      16000,
			ShippingFee:    500,
			TotalAmount:    176500,
			Details:        details,
			OrderedAt:      time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC),
			PaidAt:         paidAt,
			CanceledAt:     nil,
			ShippedAt:      nil,
			DeliveredAt:    nil,
		})
		require.NoError(t, err)
		return p
	}

	// newUC は、指定 mock を注入した usecase を生成するローカルヘルパーです。
	newUC := func(t *testing.T, cmd *mock_purchase.MockCommandService, repo *mock_purchase.MockRepository, emit *mock_outbox.MockEmitUsecase) *usecase {
		t.Helper()
		return &usecase{
			tracer: observability.NewNoopTracerFactory(t).Usecase(),
			txm:    testkit.NewMockTransactionManager(t),
			cmd:    cmd, repo: repo, emit: emit,
			clock: clocktestkit.NewMockClock(t, time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)),
		}
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ロック→所有権検証→支払い→emit→再読込を経て支払いビューを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cmd := mock_purchase.NewMockCommandService(ctrl)
			repo := mock_purchase.NewMockRepository(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)

			repo.EXPECT().LockByID(gomock.Any(), purchaseID).Return(lockable(t, userID, domainpurchase.StatusUnprocessed, nil), nil)
			repo.EXPECT().UpdatePaid(gomock.Any(), gomock.Any()).Return(nil)
			emit.EXPECT().Emit(gomock.Any(), gomock.Any()).Return(uuid.UUID{}, nil)

			paidAt := time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)
			detail := &domainpurchase.Detail{
				ID:             purchaseID,
				Code:           "pay-uc-code",
				UserID:         userID,
				StatusID:       uuidtestkit.NewTestFromSalt(t, "pay_uc_paid_status"),
				StatusName:     "支払い済み",
				SubtotalAmount: 160000,
				TaxAmount:      16000,
				ShippingFee:    500,
				TotalAmount:    176500,
				Details: []domainpurchase.PurchaseDetail{
					domainpurchase.NewPurchaseDetail(uuidtestkit.NewTestFromSalt(t, "pay_uc_d1"), domainpurchase.PurchaseDetailAttributes{
						ProductID: uuidtestkit.NewTestFromSalt(t, "pay_uc_p1"),
						Quantity:  2,
						UnitPrice: mustPrice(t, "800"),
					}),
				},
				OrderedAt: time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC),
				PaidAt:    &paidAt,
			}
			repo.EXPECT().FindDetailByID(gomock.Any(), purchaseID).Return(detail, nil)

			u := newUC(t, cmd, repo, emit)
			view, err := u.PayPurchase(context.Background(), PayPurchaseParams{PurchaseID: purchaseID, UserID: userID})
			require.NoError(t, err)
			assert.Equal(t, detail.StatusID, view.StatusID)
			assert.Equal(t, "支払い済み", view.StatusName)
			require.NotNil(t, view.PaidAt)
			assert.Equal(t, paidAt, *view.PaidAt)
			require.Len(t, view.Details, 1)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("他人の購入の場合、ErrNotFoundを返し支払い書き込みを行わない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cmd := mock_purchase.NewMockCommandService(ctrl)
			repo := mock_purchase.NewMockRepository(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)

			otherOwner := uuidtestkit.NewTestFromSalt(t, "pay_uc_other")
			repo.EXPECT().LockByID(gomock.Any(), purchaseID).Return(lockable(t, otherOwner, domainpurchase.StatusUnprocessed, nil), nil)

			u := newUC(t, cmd, repo, emit)
			_, err := u.PayPurchase(context.Background(), PayPurchaseParams{PurchaseID: purchaseID, UserID: userID})
			require.ErrorIs(t, err, apperror.ErrNotFound)
		})

		t.Run("二重支払い（既に支払い済み）の場合、ErrAlreadyPaidを伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cmd := mock_purchase.NewMockCommandService(ctrl)
			repo := mock_purchase.NewMockRepository(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)

			paidAt := time.Date(2026, time.July, 24, 0, 0, 0, 0, time.UTC)
			repo.EXPECT().LockByID(gomock.Any(), purchaseID).Return(lockable(t, userID, domainpurchase.StatusPaid, &paidAt), nil)

			u := newUC(t, cmd, repo, emit)
			_, err := u.PayPurchase(context.Background(), PayPurchaseParams{PurchaseID: purchaseID, UserID: userID})
			require.ErrorIs(t, err, domainpurchase.ErrAlreadyPaid)
		})

		t.Run("不正遷移（キャンセル済み）の場合、ErrPayNotAllowedを伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cmd := mock_purchase.NewMockCommandService(ctrl)
			repo := mock_purchase.NewMockRepository(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)

			repo.EXPECT().LockByID(gomock.Any(), purchaseID).Return(lockable(t, userID, domainpurchase.StatusCompleted, nil), nil)

			u := newUC(t, cmd, repo, emit)
			_, err := u.PayPurchase(context.Background(), PayPurchaseParams{PurchaseID: purchaseID, UserID: userID})
			require.ErrorIs(t, err, domainpurchase.ErrPayNotAllowed)
		})

		t.Run("LockPurchaseが失敗した場合はエラーを伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cmd := mock_purchase.NewMockCommandService(ctrl)
			repo := mock_purchase.NewMockRepository(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)

			repo.EXPECT().LockByID(gomock.Any(), purchaseID).Return(nil, apperror.ErrUnavailable)

			u := newUC(t, cmd, repo, emit)
			_, err := u.PayPurchase(context.Background(), PayPurchaseParams{PurchaseID: purchaseID, UserID: userID})
			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})

		t.Run("PayPurchase書き込みが失敗した場合はエラーを伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cmd := mock_purchase.NewMockCommandService(ctrl)
			repo := mock_purchase.NewMockRepository(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)

			repo.EXPECT().LockByID(gomock.Any(), purchaseID).Return(lockable(t, userID, domainpurchase.StatusUnprocessed, nil), nil)
			repo.EXPECT().UpdatePaid(gomock.Any(), gomock.Any()).Return(apperror.ErrConflict)

			u := newUC(t, cmd, repo, emit)
			_, err := u.PayPurchase(context.Background(), PayPurchaseParams{PurchaseID: purchaseID, UserID: userID})
			require.ErrorIs(t, err, apperror.ErrConflict)
		})

		t.Run("outbox発行が失敗した場合はエラーを伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cmd := mock_purchase.NewMockCommandService(ctrl)
			repo := mock_purchase.NewMockRepository(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)

			repo.EXPECT().LockByID(gomock.Any(), purchaseID).Return(lockable(t, userID, domainpurchase.StatusUnprocessed, nil), nil)
			repo.EXPECT().UpdatePaid(gomock.Any(), gomock.Any()).Return(nil)
			emit.EXPECT().Emit(gomock.Any(), gomock.Any()).Return(uuid.UUID{}, apperror.ErrInternal)

			u := newUC(t, cmd, repo, emit)
			_, err := u.PayPurchase(context.Background(), PayPurchaseParams{PurchaseID: purchaseID, UserID: userID})
			require.ErrorIs(t, err, apperror.ErrInternal)
		})

		t.Run("再読込が失敗した場合はエラーを伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cmd := mock_purchase.NewMockCommandService(ctrl)
			repo := mock_purchase.NewMockRepository(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)

			repo.EXPECT().LockByID(gomock.Any(), purchaseID).Return(lockable(t, userID, domainpurchase.StatusUnprocessed, nil), nil)
			repo.EXPECT().UpdatePaid(gomock.Any(), gomock.Any()).Return(nil)
			emit.EXPECT().Emit(gomock.Any(), gomock.Any()).Return(uuid.UUID{}, nil)
			repo.EXPECT().FindDetailByID(gomock.Any(), purchaseID).Return(nil, apperror.ErrNotFound)

			u := newUC(t, cmd, repo, emit)
			_, err := u.PayPurchase(context.Background(), PayPurchaseParams{PurchaseID: purchaseID, UserID: userID})
			require.ErrorIs(t, err, apperror.ErrNotFound)
		})
	})
}

//nolint:dupl // 支払い/キャンセルの DTO 写像テストは対称で構造の重複は不可避
func Test_toCancelPurchaseView(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("詳細読み取りモデルをキャンセルビューへ写像する", func(t *testing.T) {
			t.Parallel()

			canceledAt := time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)
			d := &domainpurchase.Detail{
				ID:             uuidtestkit.NewTestFromSalt(t, "tcv_id"),
				Code:           "tcv-code",
				UserID:         uuidtestkit.NewTestFromSalt(t, "tcv_user"),
				StatusID:       uuidtestkit.NewTestFromSalt(t, "tcv_status"),
				StatusName:     "キャンセル",
				SubtotalAmount: 160000,
				TaxAmount:      16000,
				ShippingFee:    500,
				TotalAmount:    176500,
				Details: []domainpurchase.PurchaseDetail{
					domainpurchase.NewPurchaseDetail(uuidtestkit.NewTestFromSalt(t, "tcv_d"), domainpurchase.PurchaseDetailAttributes{
						ProductID: uuidtestkit.NewTestFromSalt(t, "tcv_p"),
						Quantity:  2,
						UnitPrice: mustPrice(t, "800"),
					}),
				},
				OrderedAt:  time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC),
				CanceledAt: &canceledAt,
			}

			view := toCancelPurchaseView(d)
			assert.Equal(t, d.ID, view.ID)
			assert.Equal(t, d.Code, view.Code)
			assert.Equal(t, d.UserID, view.UserID)
			assert.Equal(t, d.StatusID, view.StatusID)
			assert.Equal(t, "キャンセル", view.StatusName)
			assert.Equal(t, d.TotalAmount, view.TotalAmount)
			require.NotNil(t, view.CanceledAt)
			assert.Equal(t, canceledAt, *view.CanceledAt)
			require.Len(t, view.Details, 1)
			assert.Equal(t, d.Details[0].ProductID(), view.Details[0].ProductID)
			assert.True(t, d.Details[0].UnitPrice().Decimal().Equal(view.Details[0].UnitPrice))
		})
	})
}

//nolint:dupl // 支払い/キャンセルの DTO 写像テストは対称で構造の重複は不可避
func Test_toPayPurchaseView(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("詳細読み取りモデルを支払いビューへ写像する", func(t *testing.T) {
			t.Parallel()

			paidAt := time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)
			d := &domainpurchase.Detail{
				ID:             uuidtestkit.NewTestFromSalt(t, "tpv_id"),
				Code:           "tpv-code",
				UserID:         uuidtestkit.NewTestFromSalt(t, "tpv_user"),
				StatusID:       uuidtestkit.NewTestFromSalt(t, "tpv_status"),
				StatusName:     "支払い済み",
				SubtotalAmount: 160000,
				TaxAmount:      16000,
				ShippingFee:    500,
				TotalAmount:    176500,
				Details: []domainpurchase.PurchaseDetail{
					domainpurchase.NewPurchaseDetail(uuidtestkit.NewTestFromSalt(t, "tpv_d"), domainpurchase.PurchaseDetailAttributes{
						ProductID: uuidtestkit.NewTestFromSalt(t, "tpv_p"),
						Quantity:  2,
						UnitPrice: mustPrice(t, "800"),
					}),
				},
				OrderedAt: time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC),
				PaidAt:    &paidAt,
			}

			view := toPayPurchaseView(d)
			assert.Equal(t, d.ID, view.ID)
			assert.Equal(t, d.Code, view.Code)
			assert.Equal(t, d.UserID, view.UserID)
			assert.Equal(t, d.StatusID, view.StatusID)
			assert.Equal(t, "支払い済み", view.StatusName)
			assert.Equal(t, d.TotalAmount, view.TotalAmount)
			require.NotNil(t, view.PaidAt)
			assert.Equal(t, paidAt, *view.PaidAt)
			require.Len(t, view.Details, 1)
			assert.Equal(t, d.Details[0].ProductID(), view.Details[0].ProductID)
			assert.True(t, d.Details[0].UnitPrice().Decimal().Equal(view.Details[0].UnitPrice))
		})
	})
}

func Test_usecase_ShipPurchase(t *testing.T) {
	t.Parallel()

	purchaseID := uuidtestkit.NewTestFromSalt(t, "ship_uc_id")
	ownerID := uuidtestkit.NewTestFromSalt(t, "ship_uc_owner")
	shippedAt := time.Date(2026, time.July, 26, 12, 0, 0, 0, time.UTC)

	// lockable は、repo.LockByID が返す再構築済み購入を生成するローカルヘルパーです。
	lockable := func(t *testing.T, status domainpurchase.Status, paidAt, shipped *time.Time) *domainpurchase.Purchase {
		t.Helper()
		details := []domainpurchase.PurchaseDetail{
			domainpurchase.NewPurchaseDetail(uuidtestkit.NewTestFromSalt(t, "ship_uc_d1"), domainpurchase.PurchaseDetailAttributes{
				ProductID: uuidtestkit.NewTestFromSalt(t, "ship_uc_p1"),
				Quantity:  2,
				UnitPrice: mustPrice(t, "800"),
			}),
		}
		p, err := domainpurchase.Reconstruct(purchaseID, domainpurchase.Attributes{
			Code:           "ship-uc-code",
			UserID:         ownerID,
			StatusID:       uuidtestkit.NewTestFromSalt(t, "ship_uc_status"),
			StatusCode:     status.Code(),
			SubtotalAmount: 160000,
			TaxAmount:      16000,
			ShippingFee:    500,
			TotalAmount:    176500,
			Details:        details,
			OrderedAt:      time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC),
			PaidAt:         paidAt,
			CanceledAt:     nil,
			ShippedAt:      shipped,
			DeliveredAt:    nil,
		})
		require.NoError(t, err)
		return p
	}

	// paidLockable は、発送可能な支払い済み購入を返すローカルヘルパーです。
	paidLockable := func(t *testing.T) *domainpurchase.Purchase {
		t.Helper()
		paidAt := time.Date(2026, time.July, 25, 0, 0, 0, 0, time.UTC)
		return lockable(t, domainpurchase.StatusPaid, &paidAt, nil)
	}

	// shippedDetail は、発送後の再読込で返す詳細読み取りモデルを生成するローカルヘルパーです。
	shippedDetail := func(t *testing.T) *domainpurchase.Detail {
		t.Helper()
		return &domainpurchase.Detail{
			ID:             purchaseID,
			Code:           "ship-uc-code",
			UserID:         ownerID,
			StatusID:       uuidtestkit.NewTestFromSalt(t, "ship_uc_shipped_status"),
			StatusName:     "発送済み",
			SubtotalAmount: 160000,
			TaxAmount:      16000,
			ShippingFee:    500,
			TotalAmount:    176500,
			Details: []domainpurchase.PurchaseDetail{
				domainpurchase.NewPurchaseDetail(uuidtestkit.NewTestFromSalt(t, "ship_uc_d1"), domainpurchase.PurchaseDetailAttributes{
					ProductID: uuidtestkit.NewTestFromSalt(t, "ship_uc_p1"),
					Quantity:  2,
					UnitPrice: mustPrice(t, "800"),
				}),
			},
			OrderedAt: time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC),
			ShippedAt: &shippedAt,
		}
	}

	// newUC は、指定 mock を注入した usecase を生成するローカルヘルパーです。
	newUC := func(t *testing.T, repo *mock_purchase.MockRepository, emit *mock_outbox.MockEmitUsecase, authorizer *mock_authz.MockAuthorizer) *usecase {
		t.Helper()
		return &usecase{
			tracer: observability.NewNoopTracerFactory(t).Usecase(),
			txm:    testkit.NewMockTransactionManager(t),
			repo:   repo, emit: emit, authorizer: authorizer,
			clock: clocktestkit.NewMockClock(t, shippedAt),
		}
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認可→ロック→発送→emit→再読込を経て発送ビューを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mock_purchase.NewMockRepository(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)
			authorizer := mock_authz.NewMockAuthorizer(ctrl)

			authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			repo.EXPECT().LockByID(gomock.Any(), purchaseID).Return(paidLockable(t), nil)
			repo.EXPECT().UpdateShipped(gomock.Any(), gomock.Any()).Return(nil)
			emit.EXPECT().Emit(gomock.Any(), gomock.Any()).Return(uuid.UUID{}, nil)
			detail := shippedDetail(t)
			repo.EXPECT().FindDetailByID(gomock.Any(), purchaseID).Return(detail, nil)

			u := newUC(t, repo, emit, authorizer)
			view, err := u.ShipPurchase(context.Background(), &auth.Authn{}, purchaseID)
			require.NoError(t, err)
			assert.Equal(t, detail.StatusID, view.StatusID)
			assert.Equal(t, "発送済み", view.StatusName)
			require.NotNil(t, view.ShippedAt)
			assert.Equal(t, shippedAt, *view.ShippedAt)
			require.Len(t, view.Details, 1)
		})

		t.Run("購入の所有者と異なる認証主体でも発送でき所有権を問わない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mock_purchase.NewMockRepository(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)
			authorizer := mock_authz.NewMockAuthorizer(ctrl)

			authn, err := auth.New("ship-uc-subject", "ship-uc-issuer", nil, nil)
			require.NoError(t, err)
			other, err := authn.WithUserID(uuidtestkit.NewTestFromSalt(t, "ship_uc_other"))
			require.NoError(t, err)

			authorizer.EXPECT().Authorize(gomock.Any(), other, gomock.Any(), gomock.Any()).Return(nil)
			repo.EXPECT().LockByID(gomock.Any(), purchaseID).Return(paidLockable(t), nil)
			repo.EXPECT().UpdateShipped(gomock.Any(), gomock.Any()).Return(nil)
			emit.EXPECT().Emit(gomock.Any(), gomock.Any()).Return(uuid.UUID{}, nil)
			repo.EXPECT().FindDetailByID(gomock.Any(), purchaseID).Return(shippedDetail(t), nil)

			u := newUC(t, repo, emit, authorizer)
			view, err := u.ShipPurchase(context.Background(), other, purchaseID)
			require.NoError(t, err)
			// 所有者は購入側の userID のまま（認証主体で上書きされない）。
			assert.Equal(t, ownerID, view.UserID)
		})

		t.Run("発送操作を所有者なしリソースとして認可する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mock_purchase.NewMockRepository(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)
			authorizer := mock_authz.NewMockAuthorizer(ctrl)

			var capturedAction authz.Action
			var capturedResource *authz.Resource
			authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ *auth.Authn, action authz.Action, resource *authz.Resource) error {
					capturedAction = action
					capturedResource = resource
					return nil
				})
			repo.EXPECT().LockByID(gomock.Any(), purchaseID).Return(paidLockable(t), nil)
			repo.EXPECT().UpdateShipped(gomock.Any(), gomock.Any()).Return(nil)
			emit.EXPECT().Emit(gomock.Any(), gomock.Any()).Return(uuid.UUID{}, nil)
			repo.EXPECT().FindDetailByID(gomock.Any(), purchaseID).Return(shippedDetail(t), nil)

			u := newUC(t, repo, emit, authorizer)
			_, err := u.ShipPurchase(context.Background(), &auth.Authn{}, purchaseID)
			require.NoError(t, err)
			assert.Equal(t, authz.ActionPurchaseShip, capturedAction)
			require.NotNil(t, capturedResource)
			assert.Equal(t, "purchase", capturedResource.Kind())
			assert.Nil(t, capturedResource.OwnerID())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("authnがnilの場合、ErrUnauthenticatedを返し認可を行わない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mock_purchase.NewMockRepository(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)
			authorizer := mock_authz.NewMockAuthorizer(ctrl)

			u := newUC(t, repo, emit, authorizer)
			_, err := u.ShipPurchase(context.Background(), nil, purchaseID)
			require.ErrorIs(t, err, apperror.ErrUnauthenticated)
		})

		t.Run("非adminの場合、認可エラーを伝播しロックを行わない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mock_purchase.NewMockRepository(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)
			authorizer := mock_authz.NewMockAuthorizer(ctrl)

			authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(authz.ErrForbidden)

			u := newUC(t, repo, emit, authorizer)
			_, err := u.ShipPurchase(context.Background(), &auth.Authn{}, purchaseID)
			require.ErrorIs(t, err, apperror.ErrPermissionDenied)
		})

		t.Run("二重発送（既に発送済み）の場合、ErrAlreadyShippedを伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mock_purchase.NewMockRepository(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)
			authorizer := mock_authz.NewMockAuthorizer(ctrl)

			paidAt := time.Date(2026, time.July, 25, 0, 0, 0, 0, time.UTC)
			authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			repo.EXPECT().LockByID(gomock.Any(), purchaseID).
				Return(lockable(t, domainpurchase.StatusShipped, &paidAt, &shippedAt), nil)

			u := newUC(t, repo, emit, authorizer)
			_, err := u.ShipPurchase(context.Background(), &auth.Authn{}, purchaseID)
			require.ErrorIs(t, err, domainpurchase.ErrAlreadyShipped)
		})

		t.Run("不正遷移（未払い）の場合、ErrShipNotAllowedを伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mock_purchase.NewMockRepository(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)
			authorizer := mock_authz.NewMockAuthorizer(ctrl)

			authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			repo.EXPECT().LockByID(gomock.Any(), purchaseID).
				Return(lockable(t, domainpurchase.StatusUnprocessed, nil, nil), nil)

			u := newUC(t, repo, emit, authorizer)
			_, err := u.ShipPurchase(context.Background(), &auth.Authn{}, purchaseID)
			require.ErrorIs(t, err, domainpurchase.ErrShipNotAllowed)
		})

		t.Run("存在しない購入の場合、ErrNotFoundを伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mock_purchase.NewMockRepository(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)
			authorizer := mock_authz.NewMockAuthorizer(ctrl)

			authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			repo.EXPECT().LockByID(gomock.Any(), purchaseID).Return(nil, apperror.ErrNotFound)

			u := newUC(t, repo, emit, authorizer)
			_, err := u.ShipPurchase(context.Background(), &auth.Authn{}, purchaseID)
			require.ErrorIs(t, err, apperror.ErrNotFound)
		})

		t.Run("発送書き込みが失敗した場合はエラーを伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mock_purchase.NewMockRepository(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)
			authorizer := mock_authz.NewMockAuthorizer(ctrl)

			authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			repo.EXPECT().LockByID(gomock.Any(), purchaseID).Return(paidLockable(t), nil)
			repo.EXPECT().UpdateShipped(gomock.Any(), gomock.Any()).Return(apperror.ErrConflict)

			u := newUC(t, repo, emit, authorizer)
			_, err := u.ShipPurchase(context.Background(), &auth.Authn{}, purchaseID)
			require.ErrorIs(t, err, apperror.ErrConflict)
		})

		t.Run("outbox発行が失敗した場合はエラーを伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mock_purchase.NewMockRepository(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)
			authorizer := mock_authz.NewMockAuthorizer(ctrl)

			authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			repo.EXPECT().LockByID(gomock.Any(), purchaseID).Return(paidLockable(t), nil)
			repo.EXPECT().UpdateShipped(gomock.Any(), gomock.Any()).Return(nil)
			emit.EXPECT().Emit(gomock.Any(), gomock.Any()).Return(uuid.UUID{}, apperror.ErrInternal)

			u := newUC(t, repo, emit, authorizer)
			_, err := u.ShipPurchase(context.Background(), &auth.Authn{}, purchaseID)
			require.ErrorIs(t, err, apperror.ErrInternal)
		})

		t.Run("再読込が失敗した場合はエラーを伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mock_purchase.NewMockRepository(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)
			authorizer := mock_authz.NewMockAuthorizer(ctrl)

			authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			repo.EXPECT().LockByID(gomock.Any(), purchaseID).Return(paidLockable(t), nil)
			repo.EXPECT().UpdateShipped(gomock.Any(), gomock.Any()).Return(nil)
			emit.EXPECT().Emit(gomock.Any(), gomock.Any()).Return(uuid.UUID{}, nil)
			repo.EXPECT().FindDetailByID(gomock.Any(), purchaseID).Return(nil, apperror.ErrNotFound)

			u := newUC(t, repo, emit, authorizer)
			_, err := u.ShipPurchase(context.Background(), &auth.Authn{}, purchaseID)
			require.ErrorIs(t, err, apperror.ErrNotFound)
		})
	})
}

//nolint:dupl // 発送/支払いの DTO 写像テストは対称で構造の重複は不可避
func Test_toShipPurchaseView(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("詳細読み取りモデルを発送ビューへ写像する", func(t *testing.T) {
			t.Parallel()

			shippedAt := time.Date(2026, time.July, 26, 12, 0, 0, 0, time.UTC)
			d := &domainpurchase.Detail{
				ID:             uuidtestkit.NewTestFromSalt(t, "tsv_id"),
				Code:           "tsv-code",
				UserID:         uuidtestkit.NewTestFromSalt(t, "tsv_user"),
				StatusID:       uuidtestkit.NewTestFromSalt(t, "tsv_status"),
				StatusName:     "発送済み",
				SubtotalAmount: 160000,
				TaxAmount:      16000,
				ShippingFee:    500,
				TotalAmount:    176500,
				Details: []domainpurchase.PurchaseDetail{
					domainpurchase.NewPurchaseDetail(uuidtestkit.NewTestFromSalt(t, "tsv_d"), domainpurchase.PurchaseDetailAttributes{
						ProductID: uuidtestkit.NewTestFromSalt(t, "tsv_p"),
						Quantity:  2,
						UnitPrice: mustPrice(t, "800"),
					}),
				},
				OrderedAt: time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC),
				ShippedAt: &shippedAt,
			}

			view := toShipPurchaseView(d)
			assert.Equal(t, d.ID, view.ID)
			assert.Equal(t, d.Code, view.Code)
			assert.Equal(t, d.UserID, view.UserID)
			assert.Equal(t, d.StatusID, view.StatusID)
			assert.Equal(t, "発送済み", view.StatusName)
			assert.Equal(t, d.TotalAmount, view.TotalAmount)
			require.NotNil(t, view.ShippedAt)
			assert.Equal(t, shippedAt, *view.ShippedAt)
			require.Len(t, view.Details, 1)
			assert.Equal(t, d.Details[0].ProductID(), view.Details[0].ProductID)
			assert.True(t, d.Details[0].UnitPrice().Decimal().Equal(view.Details[0].UnitPrice))
		})
	})
}

func Test_usecase_DeliverPurchase(t *testing.T) {
	t.Parallel()

	purchaseID := uuidtestkit.NewTestFromSalt(t, "dlv_uc_id")
	ownerID := uuidtestkit.NewTestFromSalt(t, "dlv_uc_owner")
	paidAt := time.Date(2026, time.July, 25, 0, 0, 0, 0, time.UTC)
	shippedAt := time.Date(2026, time.July, 26, 12, 0, 0, 0, time.UTC)
	deliveredAt := time.Date(2026, time.July, 28, 9, 0, 0, 0, time.UTC)

	// lockable は、repo.LockByID が返す再構築済み購入を生成するローカルヘルパーです。
	lockable := func(t *testing.T, status domainpurchase.Status, paid, shipped, delivered *time.Time) *domainpurchase.Purchase {
		t.Helper()
		details := []domainpurchase.PurchaseDetail{
			domainpurchase.NewPurchaseDetail(uuidtestkit.NewTestFromSalt(t, "dlv_uc_d1"), domainpurchase.PurchaseDetailAttributes{
				ProductID: uuidtestkit.NewTestFromSalt(t, "dlv_uc_p1"),
				Quantity:  2,
				UnitPrice: mustPrice(t, "800"),
			}),
		}
		p, err := domainpurchase.Reconstruct(purchaseID, domainpurchase.Attributes{
			Code:           "dlv-uc-code",
			UserID:         ownerID,
			StatusID:       uuidtestkit.NewTestFromSalt(t, "dlv_uc_status"),
			StatusCode:     status.Code(),
			SubtotalAmount: 160000,
			TaxAmount:      16000,
			ShippingFee:    500,
			TotalAmount:    176500,
			Details:        details,
			OrderedAt:      time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC),
			PaidAt:         paid,
			CanceledAt:     nil,
			ShippedAt:      shipped,
			DeliveredAt:    delivered,
		})
		require.NoError(t, err)
		return p
	}

	// shippedLockable は、配達可能な発送済み購入を返すローカルヘルパーです。
	shippedLockable := func(t *testing.T) *domainpurchase.Purchase {
		t.Helper()
		return lockable(t, domainpurchase.StatusShipped, &paidAt, &shippedAt, nil)
	}

	// deliveredDetail は、配達完了後の再読込で返す詳細読み取りモデルを生成するローカルヘルパーです。
	deliveredDetail := func(t *testing.T) *domainpurchase.Detail {
		t.Helper()
		return &domainpurchase.Detail{
			ID:             purchaseID,
			Code:           "dlv-uc-code",
			UserID:         ownerID,
			StatusID:       uuidtestkit.NewTestFromSalt(t, "dlv_uc_delivered_status"),
			StatusName:     "配達済み",
			SubtotalAmount: 160000,
			TaxAmount:      16000,
			ShippingFee:    500,
			TotalAmount:    176500,
			Details: []domainpurchase.PurchaseDetail{
				domainpurchase.NewPurchaseDetail(uuidtestkit.NewTestFromSalt(t, "dlv_uc_d1"), domainpurchase.PurchaseDetailAttributes{
					ProductID: uuidtestkit.NewTestFromSalt(t, "dlv_uc_p1"),
					Quantity:  2,
					UnitPrice: mustPrice(t, "800"),
				}),
			},
			OrderedAt:   time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC),
			DeliveredAt: &deliveredAt,
		}
	}

	// newUC は、指定 mock を注入した usecase を生成するローカルヘルパーです。
	newUC := func(t *testing.T, repo *mock_purchase.MockRepository, emit *mock_outbox.MockEmitUsecase, authorizer *mock_authz.MockAuthorizer) *usecase {
		t.Helper()
		return &usecase{
			tracer: observability.NewNoopTracerFactory(t).Usecase(),
			txm:    testkit.NewMockTransactionManager(t),
			repo:   repo, emit: emit, authorizer: authorizer,
			clock: clocktestkit.NewMockClock(t, deliveredAt),
		}
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認可→ロック→配達→emit→再読込を経て配達ビューを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mock_purchase.NewMockRepository(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)
			authorizer := mock_authz.NewMockAuthorizer(ctrl)

			authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			repo.EXPECT().LockByID(gomock.Any(), purchaseID).Return(shippedLockable(t), nil)
			repo.EXPECT().UpdateDelivered(gomock.Any(), gomock.Any()).Return(nil)
			emit.EXPECT().Emit(gomock.Any(), gomock.Any()).Return(uuid.UUID{}, nil)
			detail := deliveredDetail(t)
			repo.EXPECT().FindDetailByID(gomock.Any(), purchaseID).Return(detail, nil)

			u := newUC(t, repo, emit, authorizer)
			view, err := u.DeliverPurchase(context.Background(), &auth.Authn{}, purchaseID)
			require.NoError(t, err)
			assert.Equal(t, detail.StatusID, view.StatusID)
			assert.Equal(t, "配達済み", view.StatusName)
			require.NotNil(t, view.DeliveredAt)
			assert.Equal(t, deliveredAt, *view.DeliveredAt)
			require.Len(t, view.Details, 1)
		})

		t.Run("配達完了イベントをpurchase.delivered.v1として同一tx内で発行する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mock_purchase.NewMockRepository(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)
			authorizer := mock_authz.NewMockAuthorizer(ctrl)

			var captured outbox.EmitInput
			authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			repo.EXPECT().LockByID(gomock.Any(), purchaseID).Return(shippedLockable(t), nil)
			repo.EXPECT().UpdateDelivered(gomock.Any(), gomock.Any()).Return(nil)
			emit.EXPECT().Emit(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, in outbox.EmitInput) (uuid.UUID, error) {
					captured = in
					return uuid.UUID{}, nil
				})
			repo.EXPECT().FindDetailByID(gomock.Any(), purchaseID).Return(deliveredDetail(t), nil)

			// 更新と emit が 1 つのトランザクションに収まることを、Do の呼び出し回数で固定する。
			singleTx := mock_tx.NewMockManager(ctrl)
			singleTx.EXPECT().Do(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
				func(ctx context.Context, fn func(ctx context.Context) error) error { return fn(ctx) })

			u := newUC(t, repo, emit, authorizer)
			u.txm = singleTx
			_, err := u.DeliverPurchase(context.Background(), &auth.Authn{}, purchaseID)
			require.NoError(t, err)
			assert.Equal(t, event.TypeDelivered, captured.EventType)
			assert.Equal(t, aggregateType, captured.AggregateType)
			assert.Equal(t, purchaseID.String(), captured.AggregateID)

			var payload struct {
				PurchaseID  string `json:"purchaseId"`
				UserID      string `json:"userId"`
				StatusCode  int    `json:"statusCode"`
				DeliveredAt string `json:"deliveredAt"`
			}
			require.NoError(t, json.Unmarshal(captured.Payload, &payload))
			assert.Equal(t, purchaseID.String(), payload.PurchaseID)
			assert.Equal(t, domainpurchase.StatusDelivered.Code(), payload.StatusCode)
			assert.Equal(t, deliveredAt.Format(time.RFC3339Nano), payload.DeliveredAt)
		})

		t.Run("購入の所有者と異なる認証主体でも配達でき所有権を問わない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mock_purchase.NewMockRepository(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)
			authorizer := mock_authz.NewMockAuthorizer(ctrl)

			authn, err := auth.New("dlv-uc-subject", "dlv-uc-issuer", nil, nil)
			require.NoError(t, err)
			other, err := authn.WithUserID(uuidtestkit.NewTestFromSalt(t, "dlv_uc_other"))
			require.NoError(t, err)

			authorizer.EXPECT().Authorize(gomock.Any(), other, gomock.Any(), gomock.Any()).Return(nil)
			repo.EXPECT().LockByID(gomock.Any(), purchaseID).Return(shippedLockable(t), nil)
			repo.EXPECT().UpdateDelivered(gomock.Any(), gomock.Any()).Return(nil)
			emit.EXPECT().Emit(gomock.Any(), gomock.Any()).Return(uuid.UUID{}, nil)
			repo.EXPECT().FindDetailByID(gomock.Any(), purchaseID).Return(deliveredDetail(t), nil)

			u := newUC(t, repo, emit, authorizer)
			view, err := u.DeliverPurchase(context.Background(), other, purchaseID)
			require.NoError(t, err)
			// 所有者は購入側の userID のまま（認証主体で上書きされない）。
			assert.Equal(t, ownerID, view.UserID)
		})

		t.Run("配達完了操作を所有者なしリソースとして認可する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mock_purchase.NewMockRepository(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)
			authorizer := mock_authz.NewMockAuthorizer(ctrl)

			var capturedAction authz.Action
			var capturedResource *authz.Resource
			authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ *auth.Authn, action authz.Action, resource *authz.Resource) error {
					capturedAction = action
					capturedResource = resource
					return nil
				})
			repo.EXPECT().LockByID(gomock.Any(), purchaseID).Return(shippedLockable(t), nil)
			repo.EXPECT().UpdateDelivered(gomock.Any(), gomock.Any()).Return(nil)
			emit.EXPECT().Emit(gomock.Any(), gomock.Any()).Return(uuid.UUID{}, nil)
			repo.EXPECT().FindDetailByID(gomock.Any(), purchaseID).Return(deliveredDetail(t), nil)

			u := newUC(t, repo, emit, authorizer)
			_, err := u.DeliverPurchase(context.Background(), &auth.Authn{}, purchaseID)
			require.NoError(t, err)
			assert.Equal(t, authz.ActionPurchaseDeliver, capturedAction)
			require.NotNil(t, capturedResource)
			assert.Equal(t, "purchase", capturedResource.Kind())
			assert.Nil(t, capturedResource.OwnerID())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("authnがnilの場合、ErrUnauthenticatedを返し認可を行わない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mock_purchase.NewMockRepository(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)
			authorizer := mock_authz.NewMockAuthorizer(ctrl)

			u := newUC(t, repo, emit, authorizer)
			_, err := u.DeliverPurchase(context.Background(), nil, purchaseID)
			require.ErrorIs(t, err, apperror.ErrUnauthenticated)
		})

		t.Run("非adminの場合、認可エラーを伝播しロックを行わない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mock_purchase.NewMockRepository(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)
			authorizer := mock_authz.NewMockAuthorizer(ctrl)

			authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(authz.ErrForbidden)

			u := newUC(t, repo, emit, authorizer)
			_, err := u.DeliverPurchase(context.Background(), &auth.Authn{}, purchaseID)
			require.ErrorIs(t, err, apperror.ErrPermissionDenied)
		})

		t.Run("二重配達（既に配達済み）の場合、ErrAlreadyDeliveredを伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mock_purchase.NewMockRepository(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)
			authorizer := mock_authz.NewMockAuthorizer(ctrl)

			authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			repo.EXPECT().LockByID(gomock.Any(), purchaseID).
				Return(lockable(t, domainpurchase.StatusDelivered, &paidAt, &shippedAt, &deliveredAt), nil)

			u := newUC(t, repo, emit, authorizer)
			_, err := u.DeliverPurchase(context.Background(), &auth.Authn{}, purchaseID)
			require.ErrorIs(t, err, domainpurchase.ErrAlreadyDelivered)
		})

		t.Run("不正遷移（未発送の支払い済み）の場合、ErrDeliverNotAllowedを伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mock_purchase.NewMockRepository(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)
			authorizer := mock_authz.NewMockAuthorizer(ctrl)

			authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			repo.EXPECT().LockByID(gomock.Any(), purchaseID).
				Return(lockable(t, domainpurchase.StatusPaid, &paidAt, nil, nil), nil)

			u := newUC(t, repo, emit, authorizer)
			_, err := u.DeliverPurchase(context.Background(), &auth.Authn{}, purchaseID)
			require.ErrorIs(t, err, domainpurchase.ErrDeliverNotAllowed)
		})

		t.Run("存在しない購入の場合、ErrNotFoundを伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mock_purchase.NewMockRepository(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)
			authorizer := mock_authz.NewMockAuthorizer(ctrl)

			authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			repo.EXPECT().LockByID(gomock.Any(), purchaseID).Return(nil, apperror.ErrNotFound)

			u := newUC(t, repo, emit, authorizer)
			_, err := u.DeliverPurchase(context.Background(), &auth.Authn{}, purchaseID)
			require.ErrorIs(t, err, apperror.ErrNotFound)
		})

		t.Run("配達書き込みが失敗した場合はエラーを伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mock_purchase.NewMockRepository(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)
			authorizer := mock_authz.NewMockAuthorizer(ctrl)

			authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			repo.EXPECT().LockByID(gomock.Any(), purchaseID).Return(shippedLockable(t), nil)
			repo.EXPECT().UpdateDelivered(gomock.Any(), gomock.Any()).Return(apperror.ErrConflict)

			u := newUC(t, repo, emit, authorizer)
			_, err := u.DeliverPurchase(context.Background(), &auth.Authn{}, purchaseID)
			require.ErrorIs(t, err, apperror.ErrConflict)
		})

		t.Run("outbox発行が失敗した場合はtxを中断しエラーを伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mock_purchase.NewMockRepository(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)
			authorizer := mock_authz.NewMockAuthorizer(ctrl)

			authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			repo.EXPECT().LockByID(gomock.Any(), purchaseID).Return(shippedLockable(t), nil)
			repo.EXPECT().UpdateDelivered(gomock.Any(), gomock.Any()).Return(nil)
			emit.EXPECT().Emit(gomock.Any(), gomock.Any()).Return(uuid.UUID{}, apperror.ErrInternal)
			// FindDetailByID を EXPECT しないことで、emit 失敗時に tx 関数が即座に中断する
			// （＝ランナーがロールバックする）ことを担保する。

			u := newUC(t, repo, emit, authorizer)
			_, err := u.DeliverPurchase(context.Background(), &auth.Authn{}, purchaseID)
			require.ErrorIs(t, err, apperror.ErrInternal)
		})

		t.Run("再読込が失敗した場合はエラーを伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mock_purchase.NewMockRepository(ctrl)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)
			authorizer := mock_authz.NewMockAuthorizer(ctrl)

			authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			repo.EXPECT().LockByID(gomock.Any(), purchaseID).Return(shippedLockable(t), nil)
			repo.EXPECT().UpdateDelivered(gomock.Any(), gomock.Any()).Return(nil)
			emit.EXPECT().Emit(gomock.Any(), gomock.Any()).Return(uuid.UUID{}, nil)
			repo.EXPECT().FindDetailByID(gomock.Any(), purchaseID).Return(nil, apperror.ErrNotFound)

			u := newUC(t, repo, emit, authorizer)
			_, err := u.DeliverPurchase(context.Background(), &auth.Authn{}, purchaseID)
			require.ErrorIs(t, err, apperror.ErrNotFound)
		})
	})
}

//nolint:dupl // 配達/発送の DTO 写像テストは対称で構造の重複は不可避
func Test_toDeliverPurchaseView(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("詳細読み取りモデルを配達ビューへ写像する", func(t *testing.T) {
			t.Parallel()

			deliveredAt := time.Date(2026, time.July, 28, 9, 0, 0, 0, time.UTC)
			d := &domainpurchase.Detail{
				ID:             uuidtestkit.NewTestFromSalt(t, "tdv_id"),
				Code:           "tdv-code",
				UserID:         uuidtestkit.NewTestFromSalt(t, "tdv_user"),
				StatusID:       uuidtestkit.NewTestFromSalt(t, "tdv_status"),
				StatusName:     "配達済み",
				SubtotalAmount: 160000,
				TaxAmount:      16000,
				ShippingFee:    500,
				TotalAmount:    176500,
				Details: []domainpurchase.PurchaseDetail{
					domainpurchase.NewPurchaseDetail(uuidtestkit.NewTestFromSalt(t, "tdv_d"), domainpurchase.PurchaseDetailAttributes{
						ProductID: uuidtestkit.NewTestFromSalt(t, "tdv_p"),
						Quantity:  2,
						UnitPrice: mustPrice(t, "800"),
					}),
				},
				OrderedAt:   time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC),
				DeliveredAt: &deliveredAt,
			}

			view := toDeliverPurchaseView(d)
			assert.Equal(t, d.ID, view.ID)
			assert.Equal(t, d.Code, view.Code)
			assert.Equal(t, d.UserID, view.UserID)
			assert.Equal(t, d.StatusID, view.StatusID)
			assert.Equal(t, "配達済み", view.StatusName)
			assert.Equal(t, d.TotalAmount, view.TotalAmount)
			require.NotNil(t, view.DeliveredAt)
			assert.Equal(t, deliveredAt, *view.DeliveredAt)
			require.Len(t, view.Details, 1)
			assert.Equal(t, d.Details[0].ProductID(), view.Details[0].ProductID)
			assert.True(t, d.Details[0].UnitPrice().Decimal().Equal(view.Details[0].UnitPrice))
		})
	})
}
