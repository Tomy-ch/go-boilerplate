package outbox_test

import (
	"context"
	"testing"

	"go-boilerplate/internal/observability"
	mock_outbox "go-boilerplate/internal/usecase/boundary/outbox/mock"
	"go-boilerplate/internal/usecase/outbox"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNewReplay(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("依存を渡すと非nilのReplayUsecaseを生成する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_outbox.NewMockStore(ctrl)

			got := outbox.NewReplay(store, observability.NewNoopTracerFactory(t))

			assert.NotNil(t, got)
		})
	})
}

func Test_replayUsecase_ReplayDead(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("messageID が nil なら全 dead 行を pending へ戻し件数を返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_outbox.NewMockStore(ctrl)

			store.EXPECT().ReplayDead(gomock.Any(), (*uuid.UUID)(nil)).Return(int64(3), nil)

			got, err := outbox.NewReplay(store, observability.NewNoopTracerFactory(t)).
				ReplayDead(context.Background(), nil)

			require.NoError(t, err)
			assert.Equal(t, int64(3), got)
		})

		t.Run("messageID 指定時は当該 message_id のみを対象とする", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_outbox.NewMockStore(ctrl)
			id := uuid.NewTestFromSalt(t, "dead")

			store.EXPECT().ReplayDead(gomock.Any(), &id).Return(int64(1), nil)

			got, err := outbox.NewReplay(store, observability.NewNoopTracerFactory(t)).
				ReplayDead(context.Background(), &id)

			require.NoError(t, err)
			assert.Equal(t, int64(1), got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Store.ReplayDead のエラーを伝播する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_outbox.NewMockStore(ctrl)
			wantErr := xerrors.New("replay failed")

			store.EXPECT().ReplayDead(gomock.Any(), gomock.Any()).Return(int64(0), wantErr)

			_, err := outbox.NewReplay(store, observability.NewNoopTracerFactory(t)).
				ReplayDead(context.Background(), nil)

			require.ErrorIs(t, err, wantErr)
		})
	})
}
