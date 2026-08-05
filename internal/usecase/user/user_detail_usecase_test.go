package user

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/prefecture"
	mock_prefecture "go-boilerplate/internal/domain/prefecture/mock"
	mock_purchase "go-boilerplate/internal/domain/purchase/mock"
	"go-boilerplate/internal/domain/user"
	mock_user "go-boilerplate/internal/domain/user/mock"
	"go-boilerplate/internal/observability"
	authbd "go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/internal/usecase/boundary/authz"
	mock_authz "go-boilerplate/internal/usecase/boundary/authz/mock"
	clocktest "go-boilerplate/internal/usecase/boundary/clock/testkit"
	mock_tx "go-boilerplate/internal/usecase/boundary/tx/mock"
	"go-boilerplate/internal/usecase/outbox"
	mock_outbox "go-boilerplate/internal/usecase/outbox/mock"
	"go-boilerplate/internal/usecase/testkit"
	"go-boilerplate/internal/usecase/user/event"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// newAllowAuthorizer は、Authorize が常に許可（nil）を返す Authorizer モックを返します。
func newAllowAuthorizer(ctrl *gomock.Controller) *mock_authz.MockAuthorizer {
	a := mock_authz.NewMockAuthorizer(ctrl)
	a.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	return a
}

// newDenyAuthorizer は、Authorize が常に拒否（ErrForbidden）を返す Authorizer モックを返します。
func newDenyAuthorizer(ctrl *gomock.Controller) *mock_authz.MockAuthorizer {
	a := mock_authz.NewMockAuthorizer(ctrl)
	a.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(authz.ErrForbidden)
	return a
}

// newTestAuthn は、テスト用の認証主体（Authn）を返します。
func newTestAuthn(t *testing.T) *authbd.Authn {
	t.Helper()
	authn, err := authbd.New(uuidtestkit.NewTestFromSalt(t, "caller").String(), authbd.IssuerMock, nil, nil)
	require.NoError(t, err)
	return authn
}

func newActiveUser(t *testing.T, id, prefID uuid.UUID, ts time.Time) *user.User {
	t.Helper()
	u, err := user.New(id, user.Attributes{
		Profile: user.Profile{
			FirstName:    "John",
			LastName:     "Doe",
			Email:        "john@example.com",
			Phone:        "1234567890",
			PrefectureID: prefID,
			City:         "Shibuya",
			Street:       "1-2-3",
			Building:     new("Building A"),
			PostalCode:   "150-0001",
		},
		CreatedAt: ts,
		UpdatedAt: ts,
	})
	require.NoError(t, err)
	return u
}

func newUpdateDTO(prefName string) *UpdateProfileParams {
	return &UpdateProfileParams{
		FirstName: "Jane", LastName: "Smith", Email: "jane@example.com", Phone: "0987654321",
		PostalCode: "200-0002", PrefectureName: prefName, City: "Minato", Street: "4-5-6",
		Building: new("Tower"),
	}
}

func Test_usecase_GetUser(t *testing.T) {
	t.Parallel()

	authn := newTestAuthn(t)
	ctx := context.Background()
	lt := observability.NewMockUsecaseLayerTracer(t)
	now := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	id := uuidtestkit.NewTestFromSalt(t, "user")
	prefID := uuidtestkit.NewTestFromSalt(t, "prefecture")

	pft, err := prefecture.New(prefID, "Tokyo", 13)
	require.NoError(t, err)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ユーザーと都道府県名が取得できる", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			u := newActiveUser(t, id, prefID, now)

			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().FindByID(gomock.Any(), id).Return(u, nil)
			pftRepo := mock_prefecture.NewMockRepository(ctrl)
			pftRepo.EXPECT().FindByID(gomock.Any(), prefID).Return(pft, nil)

			uc := &usecase{tracer: lt, authorizer: newAllowAuthorizer(ctrl), userRepo: userRepo, pftRepo: pftRepo}
			got, err := uc.GetUser(ctx, authn, id)
			require.NoError(t, err)
			assert.Equal(t, "Tokyo", got.PrefectureName)
			assert.Equal(t, "john@example.com", got.Email)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認可が拒否される場合_ErrForbidden", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			// 認可で弾かれるため repository は呼ばれない。
			userRepo := mock_user.NewMockRepository(ctrl)

			uc := &usecase{tracer: lt, authorizer: newDenyAuthorizer(ctrl), userRepo: userRepo}
			_, err := uc.GetUser(ctx, authn, id)
			require.ErrorIs(t, err, apperror.ErrPermissionDenied)
		})

		t.Run("ユーザー取得でエラー", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			expectedErr := xerrors.New("not found")

			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().FindByID(gomock.Any(), id).Return(nil, expectedErr)

			uc := &usecase{tracer: lt, authorizer: newAllowAuthorizer(ctrl), userRepo: userRepo}
			_, err := uc.GetUser(ctx, authn, id)
			require.ErrorIs(t, err, expectedErr)
		})

		t.Run("都道府県取得でエラー", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			expectedErr := xerrors.New("prefecture not found")
			u := newActiveUser(t, id, prefID, now)

			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().FindByID(gomock.Any(), id).Return(u, nil)
			pftRepo := mock_prefecture.NewMockRepository(ctrl)
			pftRepo.EXPECT().FindByID(gomock.Any(), prefID).Return(nil, expectedErr)

			uc := &usecase{tracer: lt, authorizer: newAllowAuthorizer(ctrl), userRepo: userRepo, pftRepo: pftRepo}
			_, err := uc.GetUser(ctx, authn, id)
			require.ErrorIs(t, err, expectedErr)
		})

		t.Run("ユーザーの都道府県が NotFound の場合は参照整合性破れとして ErrInternal", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			u := newActiveUser(t, id, prefID, now)

			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().FindByID(gomock.Any(), id).Return(u, nil)
			pftRepo := mock_prefecture.NewMockRepository(ctrl)
			pftRepo.EXPECT().FindByID(gomock.Any(), prefID).Return(nil, apperror.ErrNotFound)

			uc := &usecase{tracer: lt, authorizer: newAllowAuthorizer(ctrl), userRepo: userRepo, pftRepo: pftRepo}
			_, err := uc.GetUser(ctx, authn, id)
			require.ErrorIs(t, err, apperror.ErrInternal)
		})
	})
}

func Test_usecase_UpdateUser(t *testing.T) {
	t.Parallel()

	authn := newTestAuthn(t)
	ctx := context.Background()
	lt := observability.NewMockUsecaseLayerTracer(t)
	txm := testkit.NewMockTransactionManager(t)
	now := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	id := uuidtestkit.NewTestFromSalt(t, "user")
	prefID := uuidtestkit.NewTestFromSalt(t, "prefecture")
	prefName := "Tokyo"

	pft, err := prefecture.New(prefID, prefName, 13)
	require.NoError(t, err)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("全更新が成功する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			u := newActiveUser(t, id, prefID, now)

			clock := clocktest.NewMockClockOnce(t, now)
			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().FindByID(gomock.Any(), id).Return(u, nil)
			userRepo.EXPECT().Update(gomock.Any(), gomock.AssignableToTypeOf(u)).Return(nil)
			pftRepo := mock_prefecture.NewMockRepository(ctrl)
			pftRepo.EXPECT().FindByName(gomock.Any(), prefName).Return(pft, nil)

			uc := &usecase{tracer: lt, txm: txm, clock: clock, authorizer: newAllowAuthorizer(ctrl), userRepo: userRepo, pftRepo: pftRepo}
			got, err := uc.UpdateUser(ctx, authn, id, newUpdateDTO(prefName))
			require.NoError(t, err)
			assert.Equal(t, "Jane", got.FirstName)
			assert.Equal(t, prefName, got.PrefectureName)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("対象ユーザーが存在しない", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			expectedErr := xerrors.New("not found")
			clock := clocktest.NewMockClockOnce(t, now)
			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().FindByID(gomock.Any(), id).Return(nil, expectedErr)

			uc := &usecase{tracer: lt, txm: txm, clock: clock, authorizer: newAllowAuthorizer(ctrl), userRepo: userRepo}
			_, err := uc.UpdateUser(ctx, authn, id, newUpdateDTO(prefName))
			require.ErrorIs(t, err, expectedErr)
		})

		t.Run("認可が拒否される場合_ErrForbidden", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			// 認可で弾かれるため repository / clock は呼ばれない。
			userRepo := mock_user.NewMockRepository(ctrl)

			uc := &usecase{tracer: lt, txm: txm, authorizer: newDenyAuthorizer(ctrl), userRepo: userRepo}
			_, err := uc.UpdateUser(ctx, authn, id, newUpdateDTO(prefName))
			require.ErrorIs(t, err, apperror.ErrPermissionDenied)
		})

		t.Run("都道府県解決でエラー", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			expectedErr := xerrors.New("prefecture not found")
			u := newActiveUser(t, id, prefID, now)
			clock := clocktest.NewMockClockOnce(t, now)
			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().FindByID(gomock.Any(), id).Return(u, nil)
			pftRepo := mock_prefecture.NewMockRepository(ctrl)
			pftRepo.EXPECT().FindByName(gomock.Any(), prefName).Return(nil, expectedErr)

			uc := &usecase{tracer: lt, txm: txm, clock: clock, authorizer: newAllowAuthorizer(ctrl), userRepo: userRepo, pftRepo: pftRepo}
			_, err := uc.UpdateUser(ctx, authn, id, newUpdateDTO(prefName))
			require.ErrorIs(t, err, expectedErr)
		})

		t.Run("プロフィール検証エラー", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			u := newActiveUser(t, id, prefID, now)
			clock := clocktest.NewMockClockOnce(t, now)
			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().FindByID(gomock.Any(), id).Return(u, nil)
			pftRepo := mock_prefecture.NewMockRepository(ctrl)
			pftRepo.EXPECT().FindByName(gomock.Any(), prefName).Return(pft, nil)

			dto := newUpdateDTO(prefName)
			dto.FirstName = "" // UpdateProfile の検証で失敗させる

			uc := &usecase{tracer: lt, txm: txm, clock: clock, authorizer: newAllowAuthorizer(ctrl), userRepo: userRepo, pftRepo: pftRepo}
			_, err := uc.UpdateUser(ctx, authn, id, dto)
			require.ErrorIs(t, err, user.ErrInvalidFirstName)
		})

		t.Run("永続化エラー", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			expectedErr := xerrors.New("update failed")
			u := newActiveUser(t, id, prefID, now)
			clock := clocktest.NewMockClockOnce(t, now)
			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().FindByID(gomock.Any(), id).Return(u, nil)
			userRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(expectedErr)
			pftRepo := mock_prefecture.NewMockRepository(ctrl)
			pftRepo.EXPECT().FindByName(gomock.Any(), prefName).Return(pft, nil)

			uc := &usecase{tracer: lt, txm: txm, clock: clock, authorizer: newAllowAuthorizer(ctrl), userRepo: userRepo, pftRepo: pftRepo}
			_, err := uc.UpdateUser(ctx, authn, id, newUpdateDTO(prefName))
			require.ErrorIs(t, err, expectedErr)
		})
	})
}

func Test_usecase_UpdateUserPartially(t *testing.T) {
	t.Parallel()

	authn := newTestAuthn(t)
	ctx := context.Background()
	lt := observability.NewMockUsecaseLayerTracer(t)
	txm := testkit.NewMockTransactionManager(t)
	now := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	id := uuidtestkit.NewTestFromSalt(t, "user")
	prefID := uuidtestkit.NewTestFromSalt(t, "prefecture")

	pft, err := prefecture.New(prefID, "Osaka", 27)
	require.NoError(t, err)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("都道府県指定ありの部分更新", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			u := newActiveUser(t, id, prefID, now)

			clock := clocktest.NewMockClockOnce(t, now)
			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().FindByID(gomock.Any(), id).Return(u, nil)
			userRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
			pftRepo := mock_prefecture.NewMockRepository(ctrl)
			pftRepo.EXPECT().FindByName(gomock.Any(), "Osaka").Return(pft, nil)

			uc := &usecase{tracer: lt, txm: txm, clock: clock, authorizer: newAllowAuthorizer(ctrl), userRepo: userRepo, pftRepo: pftRepo}
			dto := &PatchParamsDTO{FirstName: new("Patched"), PrefectureName: new("Osaka"), Building: new("NewTower")}
			got, err := uc.UpdateUserPartially(ctx, authn, id, dto)
			require.NoError(t, err)
			assert.Equal(t, "Patched", got.FirstName)
			assert.Equal(t, "Osaka", got.PrefectureName)
			assert.Equal(t, "NewTower", *got.Building)
		})

		t.Run("都道府県指定なしは現在値を据え置く", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			u := newActiveUser(t, id, prefID, now)

			clock := clocktest.NewMockClockOnce(t, now)
			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().FindByID(gomock.Any(), id).Return(u, nil)
			userRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
			pftRepo := mock_prefecture.NewMockRepository(ctrl)
			pftRepo.EXPECT().FindByID(gomock.Any(), prefID).Return(pft, nil)

			uc := &usecase{tracer: lt, txm: txm, clock: clock, authorizer: newAllowAuthorizer(ctrl), userRepo: userRepo, pftRepo: pftRepo}
			dto := &PatchParamsDTO{LastName: new("OnlyLast")}
			got, err := uc.UpdateUserPartially(ctx, authn, id, dto)
			require.NoError(t, err)
			assert.Equal(t, "OnlyLast", got.LastName)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("対象ユーザーが存在しない", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			expectedErr := xerrors.New("not found")
			clock := clocktest.NewMockClockOnce(t, now)
			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().FindByID(gomock.Any(), id).Return(nil, expectedErr)

			uc := &usecase{tracer: lt, txm: txm, clock: clock, authorizer: newAllowAuthorizer(ctrl), userRepo: userRepo}
			_, err := uc.UpdateUserPartially(ctx, authn, id, &PatchParamsDTO{FirstName: new("X")})
			require.ErrorIs(t, err, expectedErr)
		})

		t.Run("認可が拒否される場合_ErrForbidden", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			// 認可で弾かれるため repository / clock は呼ばれない。
			userRepo := mock_user.NewMockRepository(ctrl)

			uc := &usecase{tracer: lt, txm: txm, authorizer: newDenyAuthorizer(ctrl), userRepo: userRepo}
			_, err := uc.UpdateUserPartially(ctx, authn, id, &PatchParamsDTO{FirstName: new("X")})
			require.ErrorIs(t, err, apperror.ErrPermissionDenied)
		})

		t.Run("都道府県解決でエラー", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			expectedErr := xerrors.New("prefecture not found")
			u := newActiveUser(t, id, prefID, now)
			clock := clocktest.NewMockClockOnce(t, now)
			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().FindByID(gomock.Any(), id).Return(u, nil)
			pftRepo := mock_prefecture.NewMockRepository(ctrl)
			pftRepo.EXPECT().FindByName(gomock.Any(), "Unknown").Return(nil, expectedErr)

			uc := &usecase{tracer: lt, txm: txm, clock: clock, authorizer: newAllowAuthorizer(ctrl), userRepo: userRepo, pftRepo: pftRepo}
			_, err := uc.UpdateUserPartially(ctx, authn, id, &PatchParamsDTO{PrefectureName: new("Unknown")})
			require.ErrorIs(t, err, expectedErr)
		})

		t.Run("指定なしで現在の都道府県が NotFound の場合は ErrInternal", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			u := newActiveUser(t, id, prefID, now)
			clock := clocktest.NewMockClockOnce(t, now)
			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().FindByID(gomock.Any(), id).Return(u, nil)
			pftRepo := mock_prefecture.NewMockRepository(ctrl)
			pftRepo.EXPECT().FindByID(gomock.Any(), prefID).Return(nil, apperror.ErrNotFound)

			uc := &usecase{tracer: lt, txm: txm, clock: clock, authorizer: newAllowAuthorizer(ctrl), userRepo: userRepo, pftRepo: pftRepo}
			_, err := uc.UpdateUserPartially(ctx, authn, id, &PatchParamsDTO{FirstName: new("X")})
			require.ErrorIs(t, err, apperror.ErrInternal)
		})

		t.Run("指定なしで現在の都道府県取得が汎用エラーの場合は伝播", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			expectedErr := xerrors.New("db error")
			u := newActiveUser(t, id, prefID, now)
			clock := clocktest.NewMockClockOnce(t, now)
			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().FindByID(gomock.Any(), id).Return(u, nil)
			pftRepo := mock_prefecture.NewMockRepository(ctrl)
			pftRepo.EXPECT().FindByID(gomock.Any(), prefID).Return(nil, expectedErr)

			uc := &usecase{tracer: lt, txm: txm, clock: clock, authorizer: newAllowAuthorizer(ctrl), userRepo: userRepo, pftRepo: pftRepo}
			_, err := uc.UpdateUserPartially(ctx, authn, id, &PatchParamsDTO{FirstName: new("X")})
			require.ErrorIs(t, err, expectedErr)
		})

		t.Run("プロフィール検証エラー", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			u := newActiveUser(t, id, prefID, now)
			clock := clocktest.NewMockClockOnce(t, now)
			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().FindByID(gomock.Any(), id).Return(u, nil)
			pftRepo := mock_prefecture.NewMockRepository(ctrl)
			pftRepo.EXPECT().FindByID(gomock.Any(), prefID).Return(pft, nil)

			// 空文字へのマージで UpdateProfile（updateProfileThenSave 内）の検証を失敗させる
			dto := &PatchParamsDTO{FirstName: new("")}

			uc := &usecase{tracer: lt, txm: txm, clock: clock, authorizer: newAllowAuthorizer(ctrl), userRepo: userRepo, pftRepo: pftRepo}
			_, err := uc.UpdateUserPartially(ctx, authn, id, dto)
			require.ErrorIs(t, err, user.ErrInvalidFirstName)
		})
	})
}

func Test_usecase_DeleteUser(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	lt := observability.NewMockUsecaseLayerTracer(t)
	txm := testkit.NewMockTransactionManager(t)
	now := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	id := uuidtestkit.NewTestFromSalt(t, "user")
	prefID := uuidtestkit.NewTestFromSalt(t, "prefecture")
	authn, err := authbd.New(id.String(), authbd.IssuerMock, nil, nil)
	require.NoError(t, err)

	allowAuthorizer := func(ctrl *gomock.Controller) *mock_authz.MockAuthorizer {
		a := mock_authz.NewMockAuthorizer(ctrl)
		a.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		return a
	}

	// noInProgressPurchase は、進行中の購入を持たないユーザーを表す購入 Repository モックを返します。
	noInProgressPurchase := func(ctrl *gomock.Controller) *mock_purchase.MockRepository {
		r := mock_purchase.NewMockRepository(ctrl)
		r.EXPECT().ExistsInProgressByUserID(gomock.Any(), id).Return(false, nil)
		return r
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("論理削除と退会イベントの発行が単一トランザクションで完了する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			userLock := mock_user.NewMockLockRepository(ctrl)
			u := newActiveUser(t, id, prefID, now)

			clock := clocktest.NewMockClockOnce(t, now)
			userRepo := mock_user.NewMockRepository(ctrl)
			userLock.EXPECT().LockByID(gomock.Any(), id).Return(u, nil)
			userRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

			var emitted outbox.EmitInput
			emit := mock_outbox.NewMockEmitUsecase(ctrl)
			emit.EXPECT().Emit(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, in outbox.EmitInput) (uuid.UUID, error) {
					emitted = in
					return uuid.UUID{}, nil
				})

			// 論理削除とイベント発行を別々の tx に分けると退会だけが残るため、tx は 1 回に固定する。
			singleTx := mock_tx.NewMockManager(ctrl)
			singleTx.EXPECT().Do(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
				func(ctx context.Context, fn func(ctx context.Context) error) error { return fn(ctx) })

			uc := &usecase{
				tracer: lt, txm: singleTx, clock: clock, authorizer: allowAuthorizer(ctrl),
				userRepo: userRepo, userLock: userLock, purchaseRepo: noInProgressPurchase(ctrl), emit: emit,
			}
			require.NoError(t, uc.DeleteUser(ctx, authn, id))

			assert.Equal(t, "user", emitted.AggregateType)
			assert.Equal(t, id.String(), emitted.AggregateID)
			assert.Equal(t, event.TypeWithdrawn, emitted.EventType)

			var payload struct {
				UserID    string `json:"userId"`
				DeletedAt string `json:"deletedAt"`
			}
			require.NoError(t, json.Unmarshal(emitted.Payload, &payload))
			assert.Equal(t, id.String(), payload.UserID)
			assert.Equal(t, now.Format(time.RFC3339Nano), payload.DeletedAt)
		})

		t.Run("ユーザー行のロックを進行中購入の判定より前に取る", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			userLock := mock_user.NewMockLockRepository(ctrl)
			u := newActiveUser(t, id, prefID, now)

			clock := clocktest.NewMockClockOnce(t, now)
			userRepo := mock_user.NewMockRepository(ctrl)
			purchaseRepo := mock_purchase.NewMockRepository(ctrl)
			gomock.InOrder(
				userLock.EXPECT().LockByID(gomock.Any(), id).Return(u, nil),
				purchaseRepo.EXPECT().ExistsInProgressByUserID(gomock.Any(), id).Return(false, nil),
			)
			userRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)
			emit.EXPECT().Emit(gomock.Any(), gomock.Any()).Return(uuid.UUID{}, nil)

			uc := &usecase{
				tracer: lt, txm: txm, clock: clock, authorizer: allowAuthorizer(ctrl),
				userRepo: userRepo, userLock: userLock, purchaseRepo: purchaseRepo, emit: emit,
			}
			require.NoError(t, uc.DeleteUser(ctx, authn, id))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認可が拒否される場合_ErrForbidden", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			authorizer := mock_authz.NewMockAuthorizer(ctrl)
			authorizer.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(authz.ErrForbidden)
			// 認可で弾かれるため repository / clock は呼ばれない。
			userRepo := mock_user.NewMockRepository(ctrl)

			uc := &usecase{tracer: lt, txm: txm, authorizer: authorizer, userRepo: userRepo}
			err := uc.DeleteUser(ctx, authn, id)
			require.ErrorIs(t, err, apperror.ErrPermissionDenied)
		})

		t.Run("進行中の購入が残っている場合_ErrConflict", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			userLock := mock_user.NewMockLockRepository(ctrl)
			u := newActiveUser(t, id, prefID, now)

			clock := clocktest.NewMockClockOnce(t, now)
			userRepo := mock_user.NewMockRepository(ctrl)
			userLock.EXPECT().LockByID(gomock.Any(), id).Return(u, nil)
			// 退会を拒否するため論理削除もイベント発行も行わない。
			purchaseRepo := mock_purchase.NewMockRepository(ctrl)
			purchaseRepo.EXPECT().ExistsInProgressByUserID(gomock.Any(), id).Return(true, nil)

			uc := &usecase{
				tracer: lt, txm: txm, clock: clock, authorizer: allowAuthorizer(ctrl),
				userRepo: userRepo, userLock: userLock, purchaseRepo: purchaseRepo, emit: mock_outbox.NewMockEmitUsecase(ctrl),
			}
			err := uc.DeleteUser(ctx, authn, id)
			require.ErrorIs(t, err, apperror.ErrConflict)
			assert.Nil(t, u.DeletedAt())
		})

		t.Run("進行中の購入の確認でエラー", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			userLock := mock_user.NewMockLockRepository(ctrl)
			expectedErr := xerrors.New("purchase lookup failed")
			u := newActiveUser(t, id, prefID, now)

			clock := clocktest.NewMockClockOnce(t, now)
			userRepo := mock_user.NewMockRepository(ctrl)
			userLock.EXPECT().LockByID(gomock.Any(), id).Return(u, nil)
			purchaseRepo := mock_purchase.NewMockRepository(ctrl)
			purchaseRepo.EXPECT().ExistsInProgressByUserID(gomock.Any(), id).Return(false, expectedErr)

			uc := &usecase{
				tracer: lt, txm: txm, clock: clock, authorizer: allowAuthorizer(ctrl),
				userRepo: userRepo, userLock: userLock, purchaseRepo: purchaseRepo, emit: mock_outbox.NewMockEmitUsecase(ctrl),
			}
			require.ErrorIs(t, uc.DeleteUser(ctx, authn, id), expectedErr)
		})

		t.Run("トランザクションの実行に失敗した場合、エラーが伝播される", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			expectedErr := testkit.ExpectedDBError()

			clock := clocktest.NewMockClockOnce(t, now)
			// tx.Manager 自体が失敗する経路（接続断等）。fn は実行されないため repo 呼び出しはない。
			failingTx := mock_tx.NewMockManager(ctrl)
			failingTx.EXPECT().Do(gomock.Any(), gomock.Any()).Return(expectedErr)

			uc := &usecase{tracer: lt, txm: failingTx, clock: clock, authorizer: allowAuthorizer(ctrl)}
			require.ErrorIs(t, uc.DeleteUser(ctx, authn, id), expectedErr)
		})

		t.Run("論理削除の永続化でエラー", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			userLock := mock_user.NewMockLockRepository(ctrl)
			expectedErr := xerrors.New("update failed")
			u := newActiveUser(t, id, prefID, now)

			clock := clocktest.NewMockClockOnce(t, now)
			userRepo := mock_user.NewMockRepository(ctrl)
			userLock.EXPECT().LockByID(gomock.Any(), id).Return(u, nil)
			userRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(expectedErr)

			uc := &usecase{
				tracer: lt, txm: txm, clock: clock, authorizer: allowAuthorizer(ctrl),
				// 永続化に失敗した場合はイベントを発行しない（EXPECT を張らず未呼び出しを担保する）。
				userRepo: userRepo, userLock: userLock, purchaseRepo: noInProgressPurchase(ctrl), emit: mock_outbox.NewMockEmitUsecase(ctrl),
			}
			require.ErrorIs(t, uc.DeleteUser(ctx, authn, id), expectedErr)
		})

		t.Run("退会イベントの発行でエラー", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			userLock := mock_user.NewMockLockRepository(ctrl)
			expectedErr := xerrors.New("emit failed")
			u := newActiveUser(t, id, prefID, now)

			clock := clocktest.NewMockClockOnce(t, now)
			userRepo := mock_user.NewMockRepository(ctrl)
			userLock.EXPECT().LockByID(gomock.Any(), id).Return(u, nil)
			userRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
			emit := mock_outbox.NewMockEmitUsecase(ctrl)
			emit.EXPECT().Emit(gomock.Any(), gomock.Any()).Return(uuid.UUID{}, expectedErr)

			uc := &usecase{
				tracer: lt, txm: txm, clock: clock, authorizer: allowAuthorizer(ctrl),
				userRepo: userRepo, userLock: userLock, purchaseRepo: noInProgressPurchase(ctrl), emit: emit,
			}
			require.ErrorIs(t, uc.DeleteUser(ctx, authn, id), expectedErr)
		})

		t.Run("authnがnilの場合_ErrUnauthenticated", func(t *testing.T) {
			t.Parallel()
			// authn が nil のため認可判定以前に弾かれ、authorizer / repository は呼ばれない。
			uc := &usecase{tracer: lt, txm: txm}
			err := uc.DeleteUser(ctx, nil, id)
			require.ErrorIs(t, err, apperror.ErrUnauthenticated)
		})

		t.Run("対象ユーザーが存在しない", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			userLock := mock_user.NewMockLockRepository(ctrl)
			expectedErr := xerrors.New("not found")
			clock := clocktest.NewMockClockOnce(t, now)
			userRepo := mock_user.NewMockRepository(ctrl)
			userLock.EXPECT().LockByID(gomock.Any(), id).Return(nil, expectedErr)

			uc := &usecase{tracer: lt, txm: txm, clock: clock, authorizer: allowAuthorizer(ctrl), userRepo: userRepo, userLock: userLock}
			err := uc.DeleteUser(ctx, authn, id)
			require.ErrorIs(t, err, expectedErr)
		})

		t.Run("既に削除済みの場合_ErrAlreadyDeleted", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			userLock := mock_user.NewMockLockRepository(ctrl)
			deletedUser, err := user.New(id, user.Attributes{
				Profile: user.Profile{
					FirstName:    "John",
					LastName:     "Doe",
					Email:        "john@example.com",
					Phone:        "1234567890",
					PrefectureID: prefID,
					City:         "Shibuya",
					Street:       "1-2-3",
					Building:     new("Building A"),
					PostalCode:   "150-0001",
				},
				CreatedAt: now,
				UpdatedAt: now,
				DeletedAt: new(now),
			})
			require.NoError(t, err)

			clock := clocktest.NewMockClockOnce(t, now.Add(time.Hour))
			userRepo := mock_user.NewMockRepository(ctrl)
			userLock.EXPECT().LockByID(gomock.Any(), id).Return(deletedUser, nil)

			uc := &usecase{
				tracer: lt, txm: txm, clock: clock, authorizer: allowAuthorizer(ctrl),
				userRepo: userRepo, userLock: userLock, purchaseRepo: noInProgressPurchase(ctrl), emit: mock_outbox.NewMockEmitUsecase(ctrl),
			}
			err = uc.DeleteUser(ctx, authn, id)
			require.ErrorIs(t, err, user.ErrAlreadyDeleted)
		})
	})
}
