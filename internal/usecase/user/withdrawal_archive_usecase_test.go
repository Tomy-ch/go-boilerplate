package user

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/objectstorage"
	mock_objectstorage "go-boilerplate/internal/usecase/boundary/objectstorage/mock"
	"go-boilerplate/pkg/xerrors"
)

// testWithdrawnUserID は、退会証跡テストで使うユーザー ID です。
const testWithdrawnUserID = "3f2504e0-4f89-11d3-9a0c-0305e82c3301"

// testWithdrawalArchiveKey は、testWithdrawnUserID から決まる保存先キーです。
const testWithdrawalArchiveKey = "withdrawals/" + testWithdrawnUserID + ".json"

// newArchiveUsecase は、storage mock を差した退会証跡ユースケースを生成します。
func newArchiveUsecase(t *testing.T) (ArchiveUsecase, *mock_objectstorage.MockStorage) {
	t.Helper()

	storage := mock_objectstorage.NewMockStorage(gomock.NewController(t))
	return NewArchive(observability.NewNoopTracerFactory(t), storage), storage
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

func Test_archiveUsecase_ArchiveWithdrawal(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ユーザー ID から決まるキーへ payload をそのまま保存する", func(t *testing.T) {
			t.Parallel()

			uc, storage := newArchiveUsecase(t)
			payload := []byte(`{"userId":"` + testWithdrawnUserID + `","deletedAt":"2026-07-29T12:00:00Z"}`)
			storage.EXPECT().
				Put(gomock.Any(), objectstorage.PutObject{
					Key:         testWithdrawalArchiveKey,
					Body:        payload,
					ContentType: "application/json",
				}).
				Return(objectstorage.Path(testWithdrawalArchiveKey), nil)

			got, err := uc.ArchiveWithdrawal(t.Context(), ArchiveWithdrawalParams{
				UserID:  testWithdrawnUserID,
				Payload: payload,
			})

			require.NoError(t, err)
			assert.Equal(t, testWithdrawalArchiveKey, got)
		})

		t.Run("同じ入力で繰り返し実行しても同じ保存内容になる", func(t *testing.T) {
			t.Parallel()
			// at-least-once 配信で複数回実行されうるため、操作自体が冪等であることを固定する。
			uc, storage := newArchiveUsecase(t)
			payload := []byte(`{"userId":"` + testWithdrawnUserID + `","deletedAt":"2026-07-29T12:00:00Z"}`)
			storage.EXPECT().
				Put(gomock.Any(), objectstorage.PutObject{
					Key:         testWithdrawalArchiveKey,
					Body:        payload,
					ContentType: "application/json",
				}).
				Return(objectstorage.Path(testWithdrawalArchiveKey), nil).
				Times(2)

			params := ArchiveWithdrawalParams{UserID: testWithdrawnUserID, Payload: payload}
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

			_, err := uc.ArchiveWithdrawal(t.Context(), ArchiveWithdrawalParams{Payload: []byte(`{}`)})

			require.ErrorIs(t, err, apperror.ErrValidation)
		})

		t.Run("ユーザー ID が UUID でなければ保存しない", func(t *testing.T) {
			t.Parallel()
			// キーの一部になる値なので、区切り文字を含む値で接頭辞配下の別キーを指せないことを固定する。
			uc, _ := newArchiveUsecase(t)

			_, err := uc.ArchiveWithdrawal(t.Context(), ArchiveWithdrawalParams{
				UserID:  "../" + testWithdrawnUserID,
				Payload: []byte(`{}`),
			})

			require.ErrorIs(t, err, apperror.ErrValidation)
		})

		t.Run("payload が空なら保存しない", func(t *testing.T) {
			t.Parallel()

			uc, _ := newArchiveUsecase(t)

			_, err := uc.ArchiveWithdrawal(t.Context(), ArchiveWithdrawalParams{UserID: testWithdrawnUserID})

			require.ErrorIs(t, err, apperror.ErrValidation)
		})

		t.Run("保存の失敗をそのまま返す", func(t *testing.T) {
			t.Parallel()
			// 一時障害の分類は engine の関心なので、ここで再分類しないことを固定する。
			uc, storage := newArchiveUsecase(t)
			storage.EXPECT().
				Put(gomock.Any(), gomock.Any()).
				Return(objectstorage.Path(""), xerrors.Wrap(apperror.ErrUnavailable, "storage down"))

			_, err := uc.ArchiveWithdrawal(t.Context(), ArchiveWithdrawalParams{
				UserID:  testWithdrawnUserID,
				Payload: []byte(`{}`),
			})

			require.ErrorIs(t, err, apperror.ErrUnavailable)
		})
	})
}
