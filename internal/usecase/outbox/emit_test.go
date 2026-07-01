package outbox_test

import (
	"context"
	"encoding/json"
	"regexp"
	"testing"

	"go-boilerplate/internal/observability"
	outboxbndry "go-boilerplate/internal/usecase/boundary/outbox"
	mock_outbox "go-boilerplate/internal/usecase/boundary/outbox/mock"
	"go-boilerplate/internal/usecase/outbox"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// traceparentPattern は、W3C traceparent（00-<traceID>-<spanID>-<flags>）の形式を検証する正規表現です。
var traceparentPattern = regexp.MustCompile(`^00-[0-9a-f]{32}-[0-9a-f]{16}-[0-9a-f]{2}$`)

func TestNewEmit(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("依存を渡すと非nilのEmitUsecaseを生成する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_outbox.NewMockStore(ctrl)

			got := outbox.NewEmit(store, observability.NewNoopTracerFactory(t))

			assert.NotNil(t, got)
		})
	})
}

func TestEmitUsecase_Emit(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("noop tracer 環境で Headers が無い場合は headers を nil で INSERT し message_id を返す", func(t *testing.T) {
			t.Parallel()
			// noop tracer は有効なスパンを持たないため traceparent が注入されず、headers は nil のままになる。
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

		t.Run("noop tracer 環境で Headers がある場合はユーザ値をそのまま JSON へ marshal して INSERT する", func(t *testing.T) {
			t.Parallel()
			// noop tracer は有効なスパンを持たないため traceparent の上書きが起きず、ユーザ提供値がそのまま残る。
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

		t.Run("有効スパン環境では Headers が無くても traceparent を注入して INSERT する", func(t *testing.T) {
			t.Parallel()
			// 有効なスパンを持つ ctx では InjectTraceContextToCarrier が traceparent を headers へ載せる。
			ctx, end := observability.NewStubSpanContext(t)
			defer end()
			wantTraceID := observability.ExtractTraceContext(ctx).TraceID()

			ctrl := gomock.NewController(t)
			store := mock_outbox.NewMockStore(ctrl)
			want := uuid.NewTestFromSalt(t, "msg3")

			store.EXPECT().
				Insert(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, p outboxbndry.EmitParams) (uuid.UUID, error) {
					var h map[string]string
					require.NoError(t, json.Unmarshal(p.Headers, &h))
					require.Contains(t, h, "traceparent")
					assert.Regexp(t, traceparentPattern, h["traceparent"])
					assert.Contains(t, h["traceparent"], wantTraceID)
					return want, nil
				})

			got, err := outbox.NewEmit(store, observability.NewNoopTracerFactory(t)).
				Emit(ctx, outbox.EmitInput{
					EventType: "e.v1",
					Payload:   []byte(`{}`),
				})

			require.NoError(t, err)
			assert.Equal(t, want, got)
		})

		t.Run("有効スパン環境ではユーザ提供の traceparent を有効スパンの値で上書きしつつ他ヘッダは保持する", func(t *testing.T) {
			t.Parallel()
			// 有効なスパンがある場合、ユーザが渡した traceparent はアクティブスパンの値で上書きされ、他のヘッダは残る。
			ctx, end := observability.NewStubSpanContext(t)
			defer end()
			wantTraceID := observability.ExtractTraceContext(ctx).TraceID()

			ctrl := gomock.NewController(t)
			store := mock_outbox.NewMockStore(ctrl)
			want := uuid.NewTestFromSalt(t, "msg4")

			store.EXPECT().
				Insert(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, p outboxbndry.EmitParams) (uuid.UUID, error) {
					var h map[string]string
					require.NoError(t, json.Unmarshal(p.Headers, &h))
					require.Contains(t, h, "traceparent")
					assert.Regexp(t, traceparentPattern, h["traceparent"])
					assert.Contains(t, h["traceparent"], wantTraceID)
					assert.NotEqual(t, "00-abc", h["traceparent"])
					assert.Equal(t, "bar", h["foo"])
					return want, nil
				})

			got, err := outbox.NewEmit(store, observability.NewNoopTracerFactory(t)).
				Emit(ctx, outbox.EmitInput{
					EventType: "e.v1",
					Payload:   []byte(`{}`),
					Headers:   map[string]string{"traceparent": "00-abc", "foo": "bar"},
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
			wantErr := xerrors.New("insert failed")

			store.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(uuid.UUID{}, wantErr)

			_, err := outbox.NewEmit(store, observability.NewNoopTracerFactory(t)).
				Emit(context.Background(), outbox.EmitInput{EventType: "e.v1", Payload: []byte(`{}`)})

			require.ErrorIs(t, err, wantErr)
		})
	})
}
