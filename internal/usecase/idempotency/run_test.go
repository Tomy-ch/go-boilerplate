package idempotency_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	mock_clock "go-boilerplate/internal/usecase/boundary/clock/mock"
	idempotencybndry "go-boilerplate/internal/usecase/boundary/idempotency"
	mock_idempotency "go-boilerplate/internal/usecase/boundary/idempotency/mock"
	mock_tx "go-boilerplate/internal/usecase/boundary/tx/mock"
	"go-boilerplate/internal/usecase/idempotency"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// statusCreated は、HTTP 201（usecase 層では net/http を import できないため定義）。
const statusCreated = 201

type payload struct {
	V string `json:"v"`
}

// fixedClock は、常に now を返す clock.Clock の生成 mock を返します。
func fixedClock(ctrl *gomock.Controller, now time.Time) *mock_clock.MockClock {
	clk := mock_clock.NewMockClock(ctrl)
	clk.EXPECT().Now().Return(now).AnyTimes()
	return clk
}

// newDeps は、tx.Manager / clock.Clock を生成 mock で組んだ Deps を返します。
// Txm.Do は業務処理をそのまま実行する素通し、Clock.Now は固定時刻を返します
// （ヘッダ無しの素通し経路では呼ばれないため、いずれも AnyTimes）。
func newDeps(ctrl *gomock.Controller, store idempotencybndry.Store) idempotency.Deps {
	txm := mock_tx.NewMockManager(ctrl)
	txm.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }).AnyTimes()

	clk := fixedClock(ctrl, time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC))

	return idempotency.Deps{Txm: txm, Store: store, Clock: clk}
}

func reqCtx(fingerprint []byte) context.Context {
	return idempotency.WithRequest(context.Background(), idempotency.Request{
		Scope:       "user-1",
		Key:         "key-1",
		Fingerprint: fingerprint,
		Method:      "POST",
		Path:        "/v1/resources",
		OperationID: "PostResources",
	})
}

func TestRun(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ヘッダ無しは素通し実行され Store を呼ばない", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_idempotency.NewMockStore(ctrl)

			res, replayed, err := idempotency.Run(context.Background(), newDeps(ctrl, store), statusCreated,
				func(context.Context) (payload, error) { return payload{V: "ok"}, nil })

			require.NoError(t, err)
			assert.False(t, replayed)
			assert.Equal(t, "ok", res.V)
		})

		t.Run("新規 claim は業務処理を実行し complete に結果を保存する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_idempotency.NewMockStore(ctrl)
			want := payload{V: "created"}
			wantJSON, _ := json.Marshal(want)

			store.EXPECT().Claim(gomock.Any(), gomock.Any()).Return(true, nil)
			store.EXPECT().Complete(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, p idempotencybndry.CompleteParams) error {
					assert.Equal(t, int32(statusCreated), p.ResponseStatus)
					assert.JSONEq(t, string(wantJSON), string(p.ResponsePayload))
					return nil
				})

			res, replayed, err := idempotency.Run(reqCtx([]byte("fp")), newDeps(ctrl, store), statusCreated,
				func(context.Context) (payload, error) { return want, nil })

			require.NoError(t, err)
			assert.False(t, replayed)
			assert.Equal(t, "created", res.V)
		})

		t.Run("completed への再送は業務処理を実行せず保存結果を replay する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_idempotency.NewMockStore(ctrl)
			stored, _ := json.Marshal(payload{V: "replayed"})
			fp := []byte("fingerprint")

			store.EXPECT().Claim(gomock.Any(), gomock.Any()).Return(false, nil)
			store.EXPECT().Get(gomock.Any(), "user-1", "key-1").Return(&idempotencybndry.Record{
				Status:          idempotencybndry.StatusCompleted,
				ResponsePayload: stored,
				Fingerprint:     fp,
			}, nil)

			res, replayed, err := idempotency.Run(reqCtx(fp), newDeps(ctrl, store), statusCreated,
				func(context.Context) (payload, error) {
					t.Fatal("業務処理は呼ばれてはならない")
					return payload{}, nil
				})

			require.NoError(t, err)
			assert.True(t, replayed)
			assert.Equal(t, "replayed", res.V)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("claimed への並行再送は 409(Conflict) を返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_idempotency.NewMockStore(ctrl)
			fp := []byte("fp")

			store.EXPECT().Claim(gomock.Any(), gomock.Any()).Return(false, nil)
			store.EXPECT().Get(gomock.Any(), "user-1", "key-1").Return(&idempotencybndry.Record{
				Status:      idempotencybndry.StatusClaimed,
				Fingerprint: fp,
			}, nil)

			_, _, err := idempotency.Run(reqCtx(fp), newDeps(ctrl, store), statusCreated,
				func(context.Context) (payload, error) { return payload{}, nil })

			require.ErrorIs(t, err, apperror.ErrConflict)
		})

		t.Run("指紋不一致(同キー別ボディ)は 422(Validation) を返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_idempotency.NewMockStore(ctrl)

			store.EXPECT().Claim(gomock.Any(), gomock.Any()).Return(false, nil)
			store.EXPECT().Get(gomock.Any(), "user-1", "key-1").Return(&idempotencybndry.Record{
				Status:      idempotencybndry.StatusCompleted,
				Fingerprint: []byte("stored-fp"),
			}, nil)

			_, _, err := idempotency.Run(reqCtx([]byte("request-fp")), newDeps(ctrl, store), statusCreated,
				func(context.Context) (payload, error) { return payload{}, nil })

			require.ErrorIs(t, err, apperror.ErrValidation)
		})

		t.Run("業務処理失敗時は complete せずエラーを伝播する(キー解放)", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_idempotency.NewMockStore(ctrl)
			bizErr := apperror.ErrInvalidArgument

			store.EXPECT().Claim(gomock.Any(), gomock.Any()).Return(true, nil)

			_, _, err := idempotency.Run(reqCtx([]byte("fp")), newDeps(ctrl, store), statusCreated,
				func(context.Context) (payload, error) { return payload{}, bizErr })

			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("ロック待ちタイムアウトは 409(Conflict) を返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_idempotency.NewMockStore(ctrl)

			store.EXPECT().Claim(gomock.Any(), gomock.Any()).Return(false, idempotencybndry.ErrLockTimeout)

			_, _, err := idempotency.Run(reqCtx([]byte("fp")), newDeps(ctrl, store), statusCreated,
				func(context.Context) (payload, error) { return payload{}, nil })

			require.ErrorIs(t, err, apperror.ErrConflict)
		})

		t.Run("claim衝突直後に行が消えたレースは 409(Conflict) を返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_idempotency.NewMockStore(ctrl)

			store.EXPECT().Claim(gomock.Any(), gomock.Any()).Return(false, nil)
			store.EXPECT().Get(gomock.Any(), "user-1", "key-1").Return(nil, nil)

			_, _, err := idempotency.Run(reqCtx([]byte("fp")), newDeps(ctrl, store), statusCreated,
				func(context.Context) (payload, error) { return payload{}, nil })

			require.ErrorIs(t, err, apperror.ErrConflict)
		})

		t.Run("保存済みペイロードが壊れていれば 500(Internal) を返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_idempotency.NewMockStore(ctrl)
			fp := []byte("fp")

			store.EXPECT().Claim(gomock.Any(), gomock.Any()).Return(false, nil)
			store.EXPECT().Get(gomock.Any(), "user-1", "key-1").Return(&idempotencybndry.Record{
				Status:          idempotencybndry.StatusCompleted,
				ResponsePayload: []byte("not-json"),
				Fingerprint:     fp,
			}, nil)

			_, _, err := idempotency.Run(reqCtx(fp), newDeps(ctrl, store), statusCreated,
				func(context.Context) (payload, error) { return payload{}, nil })

			require.ErrorIs(t, err, apperror.ErrInternal)
		})

		t.Run("Claim が想定外エラーを返せばそのまま伝播する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_idempotency.NewMockStore(ctrl)
			wantErr := apperror.ErrInternal

			store.EXPECT().Claim(gomock.Any(), gomock.Any()).Return(false, wantErr)

			_, _, err := idempotency.Run(reqCtx([]byte("fp")), newDeps(ctrl, store), statusCreated,
				func(context.Context) (payload, error) { return payload{}, nil })

			require.ErrorIs(t, err, apperror.ErrInternal)
		})
	})
}
