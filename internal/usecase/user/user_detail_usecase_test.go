package user

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/domain/prefecture"
	mock_prefecture "go-boilerplate/internal/domain/prefecture/mock"
	"go-boilerplate/internal/domain/user"
	mock_user "go-boilerplate/internal/domain/user/mock"
	"go-boilerplate/internal/observability"
	mock_clock "go-boilerplate/internal/usecase/boundary/clock/mock"
	mock_security "go-boilerplate/internal/usecase/boundary/security/mock"
	"go-boilerplate/internal/usecase/testkit"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func newActiveUser(t *testing.T, id, prefID uuid.UUID, ts time.Time) *user.User {
	t.Helper()
	u, err := user.New(
		id, "John", "Doe", "hashed_password", "john@example.com", "1234567890",
		prefID, "Shibuya", "1-2-3", ptr.To("Building A"), "150-0001", ts, ts, nil,
	)
	require.NoError(t, err)
	return u
}

func newUpdateDTO(prefName string) *UpdateParamsDTO {
	return &UpdateParamsDTO{
		FirstName: "Jane", LastName: "Smith", Email: "jane@example.com", Phone: "0987654321",
		PostalCode: "200-0002", PrefectureName: prefName, City: "Minato", Street: "4-5-6",
		Building: ptr.To("Tower"), RawPassword: "valid_password",
	}
}

func Test_usecase_GetUser(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	lt := observability.NewMockUsecaseLayerTracer(t)
	now := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.Local)
	id := uuid.NewTestFromSalt(t, "user")
	prefID := uuid.NewTestFromSalt(t, "prefecture")

	pft, err := prefecture.New(prefID, "Tokyo", 13)
	require.NoError(t, err)

	t.Run("正常系_ユーザーと都道府県名が取得できる", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		u := newActiveUser(t, id, prefID, now)

		userRepo := mock_user.NewMockRepository(ctrl)
		userRepo.EXPECT().FindByID(gomock.Any(), id).Return(u, nil)
		pftRepo := mock_prefecture.NewMockRepository(ctrl)
		pftRepo.EXPECT().FindByID(gomock.Any(), prefID).Return(pft, nil)

		uc := &usecase{tracer: lt, userRepo: userRepo, pftRepo: pftRepo}
		got, err := uc.GetUser(ctx, id)
		require.NoError(t, err)
		require.Equal(t, "Tokyo", got.PrefectureName)
		require.Equal(t, "john@example.com", got.Email)
	})

	t.Run("異常系_ユーザー取得でエラー", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		expectedErr := xerrors.New("not found")

		userRepo := mock_user.NewMockRepository(ctrl)
		userRepo.EXPECT().FindByID(gomock.Any(), id).Return(nil, expectedErr)

		uc := &usecase{tracer: lt, userRepo: userRepo}
		_, err := uc.GetUser(ctx, id)
		require.ErrorIs(t, err, expectedErr)
	})

	t.Run("異常系_都道府県取得でエラー", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		expectedErr := xerrors.New("prefecture not found")
		u := newActiveUser(t, id, prefID, now)

		userRepo := mock_user.NewMockRepository(ctrl)
		userRepo.EXPECT().FindByID(gomock.Any(), id).Return(u, nil)
		pftRepo := mock_prefecture.NewMockRepository(ctrl)
		pftRepo.EXPECT().FindByID(gomock.Any(), prefID).Return(nil, expectedErr)

		uc := &usecase{tracer: lt, userRepo: userRepo, pftRepo: pftRepo}
		_, err := uc.GetUser(ctx, id)
		require.ErrorIs(t, err, expectedErr)
	})
}

func Test_usecase_UpdateUser(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	lt := observability.NewMockUsecaseLayerTracer(t)
	txm := testkit.NewMockTransactionManager(t)
	now := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.Local)
	id := uuid.NewTestFromSalt(t, "user")
	prefID := uuid.NewTestFromSalt(t, "prefecture")
	prefName := "Tokyo"

	pft, err := prefecture.New(prefID, prefName, 13)
	require.NoError(t, err)

	t.Run("正常系_全更新が成功する", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		u := newActiveUser(t, id, prefID, now)

		clock := mock_clock.NewMockClock(ctrl)
		clock.EXPECT().Now().Return(now)
		encrypter := mock_security.NewMockEncrypter(ctrl)
		encrypter.EXPECT().Hash("valid_password").Return("new_hashed", nil)
		userRepo := mock_user.NewMockRepository(ctrl)
		userRepo.EXPECT().FindByID(gomock.Any(), id).Return(u, nil)
		userRepo.EXPECT().Update(gomock.Any(), gomock.AssignableToTypeOf(u)).Return(nil)
		pftRepo := mock_prefecture.NewMockRepository(ctrl)
		pftRepo.EXPECT().FindByName(gomock.Any(), prefName).Return(pft, nil)

		uc := &usecase{tracer: lt, txm: txm, clock: clock, encrypter: encrypter, userRepo: userRepo, pftRepo: pftRepo}
		got, err := uc.UpdateUser(ctx, id, newUpdateDTO(prefName))
		require.NoError(t, err)
		require.Equal(t, "Jane", got.FirstName)
		require.Equal(t, prefName, got.PrefectureName)
	})

	t.Run("異常系_生パスワード検証エラー", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		clock := mock_clock.NewMockClock(ctrl)
		clock.EXPECT().Now().Return(now)

		dto := newUpdateDTO(prefName)
		dto.RawPassword = "short" // MinRawPasswordLength 未満

		uc := &usecase{tracer: lt, clock: clock}
		_, err := uc.UpdateUser(ctx, id, dto)
		require.ErrorIs(t, err, user.ErrInvalidRawPassword)
	})

	t.Run("異常系_暗号化エラー", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		expectedErr := xerrors.New("hash failed")
		u := newActiveUser(t, id, prefID, now)
		clock := mock_clock.NewMockClock(ctrl)
		clock.EXPECT().Now().Return(now)
		encrypter := mock_security.NewMockEncrypter(ctrl)
		encrypter.EXPECT().Hash("valid_password").Return("", expectedErr)
		userRepo := mock_user.NewMockRepository(ctrl)
		userRepo.EXPECT().FindByID(gomock.Any(), id).Return(u, nil)
		pftRepo := mock_prefecture.NewMockRepository(ctrl)
		pftRepo.EXPECT().FindByName(gomock.Any(), prefName).Return(pft, nil)

		uc := &usecase{tracer: lt, txm: txm, clock: clock, encrypter: encrypter, userRepo: userRepo, pftRepo: pftRepo}
		_, err := uc.UpdateUser(ctx, id, newUpdateDTO(prefName))
		require.ErrorIs(t, err, expectedErr)
	})

	t.Run("異常系_対象ユーザーが存在しない", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		expectedErr := xerrors.New("not found")
		clock := mock_clock.NewMockClock(ctrl)
		clock.EXPECT().Now().Return(now)
		encrypter := mock_security.NewMockEncrypter(ctrl) // Hash は存在確認後なので呼ばれない
		userRepo := mock_user.NewMockRepository(ctrl)
		userRepo.EXPECT().FindByID(gomock.Any(), id).Return(nil, expectedErr)

		uc := &usecase{tracer: lt, txm: txm, clock: clock, encrypter: encrypter, userRepo: userRepo}
		_, err := uc.UpdateUser(ctx, id, newUpdateDTO(prefName))
		require.ErrorIs(t, err, expectedErr)
	})

	t.Run("異常系_永続化エラー", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		expectedErr := xerrors.New("update failed")
		u := newActiveUser(t, id, prefID, now)
		clock := mock_clock.NewMockClock(ctrl)
		clock.EXPECT().Now().Return(now)
		encrypter := mock_security.NewMockEncrypter(ctrl)
		encrypter.EXPECT().Hash("valid_password").Return("new_hashed", nil)
		userRepo := mock_user.NewMockRepository(ctrl)
		userRepo.EXPECT().FindByID(gomock.Any(), id).Return(u, nil)
		userRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(expectedErr)
		pftRepo := mock_prefecture.NewMockRepository(ctrl)
		pftRepo.EXPECT().FindByName(gomock.Any(), prefName).Return(pft, nil)

		uc := &usecase{tracer: lt, txm: txm, clock: clock, encrypter: encrypter, userRepo: userRepo, pftRepo: pftRepo}
		_, err := uc.UpdateUser(ctx, id, newUpdateDTO(prefName))
		require.ErrorIs(t, err, expectedErr)
	})

	t.Run("異常系_都道府県解決でエラー", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		expectedErr := xerrors.New("prefecture not found")
		u := newActiveUser(t, id, prefID, now)
		clock := mock_clock.NewMockClock(ctrl)
		clock.EXPECT().Now().Return(now)
		encrypter := mock_security.NewMockEncrypter(ctrl) // Hash は都道府県解決後なので呼ばれない
		userRepo := mock_user.NewMockRepository(ctrl)
		userRepo.EXPECT().FindByID(gomock.Any(), id).Return(u, nil)
		pftRepo := mock_prefecture.NewMockRepository(ctrl)
		pftRepo.EXPECT().FindByName(gomock.Any(), prefName).Return(nil, expectedErr)

		uc := &usecase{tracer: lt, txm: txm, clock: clock, encrypter: encrypter, userRepo: userRepo, pftRepo: pftRepo}
		_, err := uc.UpdateUser(ctx, id, newUpdateDTO(prefName))
		require.ErrorIs(t, err, expectedErr)
	})

	t.Run("異常系_プロフィール検証エラー", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		u := newActiveUser(t, id, prefID, now)
		clock := mock_clock.NewMockClock(ctrl)
		clock.EXPECT().Now().Return(now)
		encrypter := mock_security.NewMockEncrypter(ctrl)
		encrypter.EXPECT().Hash("valid_password").Return("new_hashed", nil)
		userRepo := mock_user.NewMockRepository(ctrl)
		userRepo.EXPECT().FindByID(gomock.Any(), id).Return(u, nil)
		pftRepo := mock_prefecture.NewMockRepository(ctrl)
		pftRepo.EXPECT().FindByName(gomock.Any(), prefName).Return(pft, nil)

		dto := newUpdateDTO(prefName)
		dto.FirstName = "" // UpdateProfile の検証で失敗させる

		uc := &usecase{tracer: lt, txm: txm, clock: clock, encrypter: encrypter, userRepo: userRepo, pftRepo: pftRepo}
		_, err := uc.UpdateUser(ctx, id, dto)
		require.ErrorIs(t, err, user.ErrInvalidFirstName)
	})

	t.Run("異常系_パスワード更新の検証エラー", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		u := newActiveUser(t, id, prefID, now)
		clock := mock_clock.NewMockClock(ctrl)
		clock.EXPECT().Now().Return(now)
		encrypter := mock_security.NewMockEncrypter(ctrl)
		encrypter.EXPECT().Hash("valid_password").Return("", nil) // 空ハッシュ → ChangePassword で失敗
		userRepo := mock_user.NewMockRepository(ctrl)
		userRepo.EXPECT().FindByID(gomock.Any(), id).Return(u, nil)
		pftRepo := mock_prefecture.NewMockRepository(ctrl)
		pftRepo.EXPECT().FindByName(gomock.Any(), prefName).Return(pft, nil)

		uc := &usecase{tracer: lt, txm: txm, clock: clock, encrypter: encrypter, userRepo: userRepo, pftRepo: pftRepo}
		_, err := uc.UpdateUser(ctx, id, newUpdateDTO(prefName))
		require.ErrorIs(t, err, user.ErrInvalidPasswordHash)
	})
}

func Test_usecase_UpdateUserPartially(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	lt := observability.NewMockUsecaseLayerTracer(t)
	txm := testkit.NewMockTransactionManager(t)
	now := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.Local)
	id := uuid.NewTestFromSalt(t, "user")
	prefID := uuid.NewTestFromSalt(t, "prefecture")

	pft, err := prefecture.New(prefID, "Osaka", 27)
	require.NoError(t, err)

	t.Run("正常系_都道府県指定ありの部分更新", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		u := newActiveUser(t, id, prefID, now)

		clock := mock_clock.NewMockClock(ctrl)
		clock.EXPECT().Now().Return(now)
		userRepo := mock_user.NewMockRepository(ctrl)
		userRepo.EXPECT().FindByID(gomock.Any(), id).Return(u, nil)
		userRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
		pftRepo := mock_prefecture.NewMockRepository(ctrl)
		pftRepo.EXPECT().FindByName(gomock.Any(), "Osaka").Return(pft, nil)

		uc := &usecase{tracer: lt, txm: txm, clock: clock, userRepo: userRepo, pftRepo: pftRepo}
		dto := &PatchParamsDTO{FirstName: ptr.To("Patched"), PrefectureName: ptr.To("Osaka"), Building: ptr.To("NewTower")}
		got, err := uc.UpdateUserPartially(ctx, id, dto)
		require.NoError(t, err)
		require.Equal(t, "Patched", got.FirstName)
		require.Equal(t, "Osaka", got.PrefectureName)
		require.Equal(t, "NewTower", *got.Building)
	})

	t.Run("正常系_都道府県指定なしは現在値を据え置く", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		u := newActiveUser(t, id, prefID, now)

		clock := mock_clock.NewMockClock(ctrl)
		clock.EXPECT().Now().Return(now)
		userRepo := mock_user.NewMockRepository(ctrl)
		userRepo.EXPECT().FindByID(gomock.Any(), id).Return(u, nil)
		userRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
		pftRepo := mock_prefecture.NewMockRepository(ctrl)
		pftRepo.EXPECT().FindByID(gomock.Any(), prefID).Return(pft, nil)

		uc := &usecase{tracer: lt, txm: txm, clock: clock, userRepo: userRepo, pftRepo: pftRepo}
		dto := &PatchParamsDTO{LastName: ptr.To("OnlyLast")}
		got, err := uc.UpdateUserPartially(ctx, id, dto)
		require.NoError(t, err)
		require.Equal(t, "OnlyLast", got.LastName)
	})

	t.Run("異常系_対象ユーザーが存在しない", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		expectedErr := xerrors.New("not found")
		clock := mock_clock.NewMockClock(ctrl)
		clock.EXPECT().Now().Return(now)
		userRepo := mock_user.NewMockRepository(ctrl)
		userRepo.EXPECT().FindByID(gomock.Any(), id).Return(nil, expectedErr)

		uc := &usecase{tracer: lt, txm: txm, clock: clock, userRepo: userRepo}
		_, err := uc.UpdateUserPartially(ctx, id, &PatchParamsDTO{FirstName: ptr.To("X")})
		require.ErrorIs(t, err, expectedErr)
	})

	t.Run("異常系_都道府県解決でエラー", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		expectedErr := xerrors.New("prefecture not found")
		u := newActiveUser(t, id, prefID, now)
		clock := mock_clock.NewMockClock(ctrl)
		clock.EXPECT().Now().Return(now)
		userRepo := mock_user.NewMockRepository(ctrl)
		userRepo.EXPECT().FindByID(gomock.Any(), id).Return(u, nil)
		pftRepo := mock_prefecture.NewMockRepository(ctrl)
		pftRepo.EXPECT().FindByName(gomock.Any(), "Unknown").Return(nil, expectedErr)

		uc := &usecase{tracer: lt, txm: txm, clock: clock, userRepo: userRepo, pftRepo: pftRepo}
		_, err := uc.UpdateUserPartially(ctx, id, &PatchParamsDTO{PrefectureName: ptr.To("Unknown")})
		require.ErrorIs(t, err, expectedErr)
	})

	t.Run("異常系_プロフィール検証エラー", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		u := newActiveUser(t, id, prefID, now)
		clock := mock_clock.NewMockClock(ctrl)
		clock.EXPECT().Now().Return(now)
		userRepo := mock_user.NewMockRepository(ctrl)
		userRepo.EXPECT().FindByID(gomock.Any(), id).Return(u, nil)
		pftRepo := mock_prefecture.NewMockRepository(ctrl)
		pftRepo.EXPECT().FindByID(gomock.Any(), prefID).Return(pft, nil)

		// 空文字へのマージで UpdateProfile（updateProfileThenSave 内）の検証を失敗させる
		dto := &PatchParamsDTO{FirstName: ptr.To("")}

		uc := &usecase{tracer: lt, txm: txm, clock: clock, userRepo: userRepo, pftRepo: pftRepo}
		_, err := uc.UpdateUserPartially(ctx, id, dto)
		require.ErrorIs(t, err, user.ErrInvalidFirstName)
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

	t.Run("正常系_論理削除が成功する", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		u := newActiveUser(t, id, prefID, now)

		clock := mock_clock.NewMockClock(ctrl)
		clock.EXPECT().Now().Return(now)
		userRepo := mock_user.NewMockRepository(ctrl)
		userRepo.EXPECT().FindByID(gomock.Any(), id).Return(u, nil)
		userRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

		uc := &usecase{tracer: lt, txm: txm, clock: clock, userRepo: userRepo}
		err := uc.DeleteUser(ctx, id)
		require.NoError(t, err)
	})

	t.Run("異常系_対象ユーザーが存在しない", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		expectedErr := xerrors.New("not found")
		clock := mock_clock.NewMockClock(ctrl)
		clock.EXPECT().Now().Return(now)
		userRepo := mock_user.NewMockRepository(ctrl)
		userRepo.EXPECT().FindByID(gomock.Any(), id).Return(nil, expectedErr)

		uc := &usecase{tracer: lt, txm: txm, clock: clock, userRepo: userRepo}
		err := uc.DeleteUser(ctx, id)
		require.ErrorIs(t, err, expectedErr)
	})

	t.Run("異常系_既に削除済みの場合_ErrAlreadyDeleted", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		deletedUser, err := user.New(
			id, "John", "Doe", "hashed_password", "john@example.com", "1234567890",
			prefID, "Shibuya", "1-2-3", ptr.To("Building A"), "150-0001",
			now, now, ptr.To(now),
		)
		require.NoError(t, err)

		clock := mock_clock.NewMockClock(ctrl)
		clock.EXPECT().Now().Return(now.Add(time.Hour))
		userRepo := mock_user.NewMockRepository(ctrl)
		userRepo.EXPECT().FindByID(gomock.Any(), id).Return(deletedUser, nil)

		uc := &usecase{tracer: lt, txm: txm, clock: clock, userRepo: userRepo}
		err = uc.DeleteUser(ctx, id)
		require.ErrorIs(t, err, user.ErrAlreadyDeleted)
	})
}
