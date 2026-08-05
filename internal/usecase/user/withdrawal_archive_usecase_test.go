package user_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/objectstorage"
	mock_objectstorage "go-boilerplate/internal/usecase/boundary/objectstorage/mock"
	"go-boilerplate/internal/usecase/user"
	"go-boilerplate/pkg/xerrors"
)

// newArchiveUsecase は、storage mock を差した退会証跡ユースケースを生成します。
func newArchiveUsecase(t *testing.T) (user.ArchiveUsecase, *mock_objectstorage.MockStorage) {
	t.Helper()

	storage := mock_objectstorage.NewMockStorage(gomock.NewController(t))
	return user.NewArchive(observability.NewNoopTracerFactory(t), storage), storage
}

func TestNewArchive(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("TracerFactory と Storage から ArchiveUsecase を生成する", func(t *testing.T) {
			t.Parallel()

			got, _ := newArchiveUsecase(t)

			assert.NotNil(t, got)
		})
	})
}

func TestArchiveUsecase_ArchiveWithdrawal(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ユーザー ID から決まるキーへ payload をそのまま保存する", func(t *testing.T) {
			t.Parallel()

			uc, storage := newArchiveUsecase(t)
			payload := []byte(`{"userId":"u1","deletedAt":"2026-07-29T12:00:00Z"}`)
			storage.EXPECT().
				Put(gomock.Any(), objectstorage.PutObject{
					Key:         "withdrawals/u1.json",
					Body:        payload,
					ContentType: "application/json",
				}).
				Return(objectstorage.Path("withdrawals/u1.json"), nil)

			got, err := uc.ArchiveWithdrawal(t.Context(), user.ArchiveWithdrawalParams{
				UserID:  "u1",
				Payload: payload,
			})

			require.NoError(t, err)
			assert.Equal(t, "withdrawals/u1.json", got)
		})

		t.Run("同じ入力で繰り返し実行しても同じ保存内容になる", func(t *testing.T) {
			t.Parallel()
			// at-least-once 配信で複数回実行されうるため、操作自体が冪等であることを固定する。
			uc, storage := newArchiveUsecase(t)
			payload := []byte(`{"userId":"u1","deletedAt":"2026-07-29T12:00:00Z"}`)
			storage.EXPECT().
				Put(gomock.Any(), objectstorage.PutObject{
					Key:         "withdrawals/u1.json",
					Body:        payload,
					ContentType: "application/json",
				}).
				Return(objectstorage.Path("withdrawals/u1.json"), nil).
				Times(2)

			params := user.ArchiveWithdrawalParams{UserID: "u1", Payload: payload}
			first, err := uc.ArchiveWithdrawal(t.Context(), params)
			require.NoError(t, err)
			second, err := uc.ArchiveWithdrawal(t.Context(), params)

			require.NoError(t, err)
			assert.Equal(t, first, second)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ユーザー ID が空なら保存しない", func(t *testing.T) {
			t.Parallel()

			uc, _ := newArchiveUsecase(t)

			_, err := uc.ArchiveWithdrawal(t.Context(), user.ArchiveWithdrawalParams{
				Payload: []byte(`{}`),
			})

			require.ErrorIs(t, err, apperror.ErrValidation)
		})

		t.Run("payload が空なら保存しない", func(t *testing.T) {
			t.Parallel()

			uc, _ := newArchiveUsecase(t)

			_, err := uc.ArchiveWithdrawal(t.Context(), user.ArchiveWithdrawalParams{UserID: "u1"})

			require.ErrorIs(t, err, apperror.ErrValidation)
		})

		t.Run("保存の失敗をそのまま返す", func(t *testing.T) {
			t.Parallel()
			// 一時障害の分類は engine の関心なので、ここで再分類しないことを固定する。
			uc, storage := newArchiveUsecase(t)
			storage.EXPECT().
				Put(gomock.Any(), gomock.Any()).
				Return(objectstorage.Path(""), xerrors.Wrap(apperror.ErrUnavailable, "storage down"))

			_, err := uc.ArchiveWithdrawal(t.Context(), user.ArchiveWithdrawalParams{
				UserID:  "u1",
				Payload: []byte(`{}`),
			})

			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})
	})
}
