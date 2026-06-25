package outbox_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"go-boilerplate/internal/observability"
	outboxbndry "go-boilerplate/internal/usecase/boundary/outbox"
	mock_outbox "go-boilerplate/internal/usecase/boundary/outbox/mock"
	"go-boilerplate/internal/usecase/outbox"
	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestEmitUsecase_Emit(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Headers が無い場合は headers を nil で INSERT し message_id を返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_outbox.NewMockStore(ctrl)
			want := uuid.NewTestFromSalt(t, "msg")

			store.EXPECT().
				Insert(gomock.Any(), outboxbndry.EmitParams{
					AggregateType: "Purchase",
					AggregateID:   "p-1",
					EventType:     "purchase.created.v1",
					Payload:       []byte(`{"v":1}`),
					Headers:       nil,
				}).
				Return(want, nil)

			got, err := outbox.NewEmit(store, observability.NewNoopTracerFactory(t)).
				Emit(context.Background(), outbox.EmitInput{
					AggregateType: "Purchase",
					AggregateID:   "p-1",
					EventType:     "purchase.created.v1",
					Payload:       []byte(`{"v":1}`),
				})

			require.NoError(t, err)
			assert.Equal(t, want, got)
		})

		t.Run("Headers がある場合は JSON へ marshal して INSERT する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_outbox.NewMockStore(ctrl)
			want := uuid.NewTestFromSalt(t, "msg2")

			store.EXPECT().
				Insert(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, p outboxbndry.EmitParams) (uuid.UUID, error) {
					var h map[string]string
					require.NoError(t, json.Unmarshal(p.Headers, &h))
					assert.Equal(t, map[string]string{"traceparent": "00-abc"}, h)
					return want, nil
				})

			got, err := outbox.NewEmit(store, observability.NewNoopTracerFactory(t)).
				Emit(context.Background(), outbox.EmitInput{
					EventType: "e.v1",
					Payload:   []byte(`{}`),
					Headers:   map[string]string{"traceparent": "00-abc"},
				})

			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Store.Insert のエラーを伝播する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_outbox.NewMockStore(ctrl)
			wantErr := errors.New("insert failed")

			store.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(uuid.UUID{}, wantErr)

			_, err := outbox.NewEmit(store, observability.NewNoopTracerFactory(t)).
				Emit(context.Background(), outbox.EmitInput{EventType: "e.v1", Payload: []byte(`{}`)})

			require.ErrorIs(t, err, wantErr)
		})
	})
}
