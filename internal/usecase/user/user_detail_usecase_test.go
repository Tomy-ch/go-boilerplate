package user

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/prefecture"
	mock_prefecture "go-boilerplate/internal/domain/prefecture/mock"
	"go-boilerplate/internal/domain/user"
	mock_user "go-boilerplate/internal/domain/user/mock"
	"go-boilerplate/internal/observability"
	authbd "go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/internal/usecase/boundary/authz"
	mock_authz "go-boilerplate/internal/usecase/boundary/authz/mock"
	clocktest "go-boilerplate/internal/usecase/boundary/clock/testkit"
	mock_security "go-boilerplate/internal/usecase/boundary/security/mock"
	"go-boilerplate/internal/usecase/testkit"
	"go-boilerplate/pkg/uuid"
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
	authn, err := authbd.New(uuid.NewTestFromSalt(t, "caller").String(), authbd.ProviderMock, nil, nil)
	require.NoError(t, err)
	return authn
}

func newActiveUser(t *testing.T, id, prefID uuid.UUID, ts time.Time) *user.User {
	t.Helper()
	u, err := user.New(
		id, "John", "Doe", "hashed_password", "john@example.com", "1234567890",
		prefID, "Shibuya", "1-2-3", new("Building A"), "150-0001", ts, ts, nil,
	)
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
	now := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.Local)
	id := uuid.NewTestFromSalt(t, "user")
	prefID := uuid.NewTestFromSalt(t, "prefecture")

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
	now := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.Local)
	id := uuid.NewTestFromSalt(t, "user")
	prefID := uuid.NewTestFromSalt(t, "prefecture")
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

func Test_usecase_ChangePassword(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	lt := observability.NewMockUsecaseLayerTracer(t)
	txm := testkit.NewMockTransactionManager(t)
	now := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.Local)
	id := uuid.NewTestFromSalt(t, "user")
	prefID := uuid.NewTestFromSalt(t, "prefecture")

	const (
		currentPassword = "current_password"
		newPassword     = "new_valid_password" //nolint:gosec // G101: テスト用のダミーパスワードで実際の資格情報ではない
		storedHash      = "hashed_password"    // newActiveUser が設定する passwordHash
	)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("パスワード変更が成功する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			u := newActiveUser(t, id, prefID, now)
			clock := clocktest.NewMockClockOnce(t, now)
			encrypter := mock_security.NewMockHasher(ctrl)
			encrypter.EXPECT().Compare(storedHash, currentPassword).Return(true, nil)
			encrypter.EXPECT().Hash(newPassword).Return("new_hashed", nil)
			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().FindByID(gomock.Any(), id).Return(u, nil)
			userRepo.EXPECT().Update(gomock.Any(), gomock.AssignableToTypeOf(u)).Return(nil)

			uc := &usecase{tracer: lt, txm: txm, clock: clock, encrypter: encrypter, userRepo: userRepo}
			err := uc.ChangePassword(ctx, id, currentPassword, newPassword)
			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("現パスワードの検証エラー", func(t *testing.T) {
			t.Parallel()
			clock := clocktest.NewMockClockOnce(t, now)

			uc := &usecase{tracer: lt, clock: clock}
			err := uc.ChangePassword(ctx, id, "short", newPassword)
			require.ErrorIs(t, err, user.ErrInvalidRawPassword)
		})

		t.Run("新パスワードの検証エラー", func(t *testing.T) {
			t.Parallel()
			clock := clocktest.NewMockClockOnce(t, now)

			uc := &usecase{tracer: lt, clock: clock}
			err := uc.ChangePassword(ctx, id, currentPassword, "short")
			require.ErrorIs(t, err, user.ErrInvalidRawPassword)
		})

		t.Run("対象ユーザーが存在しない", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			expectedErr := xerrors.New("not found")
			clock := clocktest.NewMockClockOnce(t, now)
			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().FindByID(gomock.Any(), id).Return(nil, expectedErr)

			uc := &usecase{tracer: lt, txm: txm, clock: clock, userRepo: userRepo}
			err := uc.ChangePassword(ctx, id, currentPassword, newPassword)
			require.ErrorIs(t, err, expectedErr)
		})

		t.Run("現パスワードが一致しない", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			u := newActiveUser(t, id, prefID, now)
			clock := clocktest.NewMockClockOnce(t, now)
			encrypter := mock_security.NewMockHasher(ctrl)
			encrypter.EXPECT().Compare(storedHash, currentPassword).Return(false, nil)
			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().FindByID(gomock.Any(), id).Return(u, nil)

			uc := &usecase{tracer: lt, txm: txm, clock: clock, encrypter: encrypter, userRepo: userRepo}
			err := uc.ChangePassword(ctx, id, currentPassword, newPassword)
			require.ErrorIs(t, err, user.ErrCurrentPasswordMismatch)
		})

		t.Run("パスワード照合でエラー", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			expectedErr := xerrors.New("compare failed")
			u := newActiveUser(t, id, prefID, now)
			clock := clocktest.NewMockClockOnce(t, now)
			encrypter := mock_security.NewMockHasher(ctrl)
			encrypter.EXPECT().Compare(storedHash, currentPassword).Return(false, expectedErr)
			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().FindByID(gomock.Any(), id).Return(u, nil)

			uc := &usecase{tracer: lt, txm: txm, clock: clock, encrypter: encrypter, userRepo: userRepo}
			err := uc.ChangePassword(ctx, id, currentPassword, newPassword)
			require.ErrorIs(t, err, expectedErr)
		})

		t.Run("新パスワードのハッシュ化エラー", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			expectedErr := xerrors.New("hash failed")
			u := newActiveUser(t, id, prefID, now)
			clock := clocktest.NewMockClockOnce(t, now)
			encrypter := mock_security.NewMockHasher(ctrl)
			encrypter.EXPECT().Compare(storedHash, currentPassword).Return(true, nil)
			encrypter.EXPECT().Hash(newPassword).Return("", expectedErr)
			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().FindByID(gomock.Any(), id).Return(u, nil)

			uc := &usecase{tracer: lt, txm: txm, clock: clock, encrypter: encrypter, userRepo: userRepo}
			err := uc.ChangePassword(ctx, id, currentPassword, newPassword)
			require.ErrorIs(t, err, expectedErr)
		})

		t.Run("ドメインのパスワード変更で検証エラー", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			u := newActiveUser(t, id, prefID, now)
			clock := clocktest.NewMockClockOnce(t, now)
			encrypter := mock_security.NewMockHasher(ctrl)
			encrypter.EXPECT().Compare(storedHash, currentPassword).Return(true, nil)
			encrypter.EXPECT().Hash(newPassword).Return("", nil) // 空ハッシュ → ドメイン ChangePassword で失敗
			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().FindByID(gomock.Any(), id).Return(u, nil)

			uc := &usecase{tracer: lt, txm: txm, clock: clock, encrypter: encrypter, userRepo: userRepo}
			err := uc.ChangePassword(ctx, id, currentPassword, newPassword)
			require.ErrorIs(t, err, user.ErrInvalidPasswordHash)
		})

		t.Run("永続化エラー", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			expectedErr := xerrors.New("update failed")
			u := newActiveUser(t, id, prefID, now)
			clock := clocktest.NewMockClockOnce(t, now)
			encrypter := mock_security.NewMockHasher(ctrl)
			encrypter.EXPECT().Compare(storedHash, currentPassword).Return(true, nil)
			encrypter.EXPECT().Hash(newPassword).Return("new_hashed", nil)
			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().FindByID(gomock.Any(), id).Return(u, nil)
			userRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(expectedErr)

			uc := &usecase{tracer: lt, txm: txm, clock: clock, encrypter: encrypter, userRepo: userRepo}
			err := uc.ChangePassword(ctx, id, currentPassword, newPassword)
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
	now := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.Local)
	id := uuid.NewTestFromSalt(t, "user")
	prefID := uuid.NewTestFromSalt(t, "prefecture")

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
	now := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.Local)
	id := uuid.NewTestFromSalt(t, "user")
	prefID := uuid.NewTestFromSalt(t, "prefecture")
	authn, err := authbd.New(id.String(), authbd.ProviderMock, nil, nil)
	require.NoError(t, err)

	allowAuthorizer := func(ctrl *gomock.Controller) *mock_authz.MockAuthorizer {
		a := mock_authz.NewMockAuthorizer(ctrl)
		a.EXPECT().Authorize(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		return a
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("論理削除が成功する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			u := newActiveUser(t, id, prefID, now)

			clock := clocktest.NewMockClockOnce(t, now)
			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().FindByID(gomock.Any(), id).Return(u, nil)
			userRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

			uc := &usecase{tracer: lt, txm: txm, clock: clock, authorizer: allowAuthorizer(ctrl), userRepo: userRepo}
			err := uc.DeleteUser(ctx, authn, id)
			require.NoError(t, err)
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
			expectedErr := xerrors.New("not found")
			clock := clocktest.NewMockClockOnce(t, now)
			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().FindByID(gomock.Any(), id).Return(nil, expectedErr)

			uc := &usecase{tracer: lt, txm: txm, clock: clock, authorizer: allowAuthorizer(ctrl), userRepo: userRepo}
			err := uc.DeleteUser(ctx, authn, id)
			require.ErrorIs(t, err, expectedErr)
		})

		t.Run("既に削除済みの場合_ErrAlreadyDeleted", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			deletedUser, err := user.New(
				id, "John", "Doe", "hashed_password", "john@example.com", "1234567890",
				prefID, "Shibuya", "1-2-3", new("Building A"), "150-0001",
				now, now, new(now),
			)
			require.NoError(t, err)

			clock := clocktest.NewMockClockOnce(t, now.Add(time.Hour))
			userRepo := mock_user.NewMockRepository(ctrl)
			userRepo.EXPECT().FindByID(gomock.Any(), id).Return(deletedUser, nil)

			uc := &usecase{tracer: lt, txm: txm, clock: clock, authorizer: allowAuthorizer(ctrl), userRepo: userRepo}
			err = uc.DeleteUser(ctx, authn, id)
			require.ErrorIs(t, err, user.ErrAlreadyDeleted)
		})
	})
}
