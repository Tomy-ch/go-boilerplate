package idempotency_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/usecase/boundary/clock/testkit"
	idempotencybndry "go-boilerplate/internal/usecase/boundary/idempotency"
	mock_idempotency "go-boilerplate/internal/usecase/boundary/idempotency/mock"
	mock_tx "go-boilerplate/internal/usecase/boundary/tx/mock"
	"go-boilerplate/internal/usecase/idempotency"
	mock_idempotencyuc "go-boilerplate/internal/usecase/idempotency/mock"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const statusCreated = 201

// fixedNow は、固定クロックが返すテスト用の基準時刻です。
var fixedNow = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

type payload struct {
	V string `json:"v"`
}

// uuidPayload は、非公開フィールドのみを持つ値オブジェクト（`uuid.UUID`）を含む DTO です。
type uuidPayload struct {
	ID uuid.UUID `json:"id"`
}

// unmarshalable は、json.Marshal が必ず失敗する（func フィールドを持つ）型です。
type unmarshalable struct {
	Fn func() `json:"fn"`
}

// newDeps は、業務 tx を 1 回だけ実行する想定の Deps を返します（Txm.Do は Times(1)）。
// 冪等処理に入る（Idempotency-Key あり）経路専用で、素通し経路では newDepsNoTx を使います。
func newDeps(t *testing.T, store idempotencybndry.Store) idempotency.Deps {
	t.Helper()
	txm := mock_tx.NewMockManager(gomock.NewController(t))
	txm.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }).Times(1)

	return idempotency.Deps{
		Txm:   txm,
		Store: store,
		Clock: testkit.NewMockClock(t, fixedNow),
	}
}

// newDepsNoTx は、Txm.Do が一度も呼ばれないこと（Times(0)）を検証するための Deps を返します。
// 素通し経路（Idempotency-Key 無し / Scope 空）で tx が使われないことの確認に用います。
func newDepsNoTx(t *testing.T, store idempotencybndry.Store) idempotency.Deps {
	t.Helper()
	txm := mock_tx.NewMockManager(gomock.NewController(t))
	txm.EXPECT().Do(gomock.Any(), gomock.Any()).Times(0)

	return idempotency.Deps{
		Txm:   txm,
		Store: store,
		Clock: testkit.NewMockClock(t, fixedNow),
	}
}

// newDepsWithMetrics は、newDeps に Metrics mock を差した Deps を返します。
func newDepsWithMetrics(
	t *testing.T, store idempotencybndry.Store, m idempotency.Metrics,
) idempotency.Deps {
	t.Helper()
	deps := newDeps(t, store)
	deps.Metrics = m
	return deps
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

	const op = "PostResources"

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ヘッダ無しは素通し実行され Store を呼ばない", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_idempotency.NewMockStore(ctrl)

			res, replayed, err := idempotency.Run(context.Background(), newDepsNoTx(t, store), statusCreated,
				func(context.Context) (payload, error) { return payload{V: "ok"}, nil })

			require.NoError(t, err)
			assert.False(t, replayed)
			assert.Equal(t, "ok", res.V)
		})

		t.Run("Request はあるが Scope 空なら素通し実行され Store を呼ばない", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_idempotency.NewMockStore(ctrl)

			ctx := idempotency.WithRequest(context.Background(), idempotency.Request{
				Key:         "key-1",
				Fingerprint: []byte("fp"),
				Method:      "POST",
				Path:        "/v1/resources",
				OperationID: "PostResources",
			})

			res, replayed, err := idempotency.Run(ctx, newDepsNoTx(t, store), statusCreated,
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

			res, replayed, err := idempotency.Run(reqCtx([]byte("fp")), newDeps(t, store), statusCreated,
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

			res, replayed, err := idempotency.Run(reqCtx(fp), newDeps(t, store), statusCreated,
				func(context.Context) (payload, error) {
					t.Fatal("業務処理は呼ばれてはならない")
					return payload{}, nil
				})

			require.NoError(t, err)
			assert.True(t, replayed)
			assert.Equal(t, "replayed", res.V)
		})

		t.Run("保存した DTO は UUID を含んでいても replay で同じ値に復元される", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_idempotency.NewMockStore(ctrl)
			want := uuidPayload{ID: uuidtestkit.NewTestFromSalt(t, "purchase-1")}
			fp := []byte("fingerprint")

			var stored []byte
			store.EXPECT().Claim(gomock.Any(), gomock.Any()).Return(true, nil)
			store.EXPECT().Complete(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, p idempotencybndry.CompleteParams) error {
					stored = p.ResponsePayload
					return nil
				})

			_, _, err := idempotency.Run(reqCtx(fp), newDeps(t, store), statusCreated,
				func(context.Context) (uuidPayload, error) { return want, nil })
			require.NoError(t, err)

			store.EXPECT().Claim(gomock.Any(), gomock.Any()).Return(false, nil)
			store.EXPECT().Get(gomock.Any(), "user-1", "key-1").Return(&idempotencybndry.Record{
				Status:          idempotencybndry.StatusCompleted,
				ResponsePayload: stored,
				Fingerprint:     fp,
			}, nil)

			res, replayed, err := idempotency.Run(reqCtx(fp), newDeps(t, store), statusCreated,
				func(context.Context) (uuidPayload, error) {
					t.Fatal("業務処理は呼ばれてはならない")
					return uuidPayload{}, nil
				})

			require.NoError(t, err)
			assert.True(t, replayed)
			assert.True(t, want.ID.Equal(res.ID))
		})

		t.Run("completed への再送は IncHit を計上する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_idempotency.NewMockStore(ctrl)
			metrics := mock_idempotencyuc.NewMockMetrics(ctrl)
			stored, _ := json.Marshal(payload{V: "replayed"})
			fp := []byte("fp")

			store.EXPECT().Claim(gomock.Any(), gomock.Any()).Return(false, nil)
			store.EXPECT().Get(gomock.Any(), "user-1", "key-1").Return(&idempotencybndry.Record{
				Status:          idempotencybndry.StatusCompleted,
				ResponsePayload: stored,
				Fingerprint:     fp,
			}, nil)
			metrics.EXPECT().IncHit(gomock.Any(), op)

			_, replayed, err := idempotency.Run(reqCtx(fp), newDepsWithMetrics(t, store, metrics), statusCreated,
				func(context.Context) (payload, error) { return payload{}, nil })

			require.NoError(t, err)
			assert.True(t, replayed)
		})

		t.Run("新規 claim は IncMiss を計上する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_idempotency.NewMockStore(ctrl)
			metrics := mock_idempotencyuc.NewMockMetrics(ctrl)

			store.EXPECT().Claim(gomock.Any(), gomock.Any()).Return(true, nil)
			store.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(nil)
			metrics.EXPECT().IncMiss(gomock.Any(), op)

			_, replayed, err := idempotency.Run(reqCtx([]byte("fp")), newDepsWithMetrics(t, store, metrics), statusCreated,
				func(context.Context) (payload, error) { return payload{V: "created"}, nil })

			require.NoError(t, err)
			assert.False(t, replayed)
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

			_, _, err := idempotency.Run(reqCtx(fp), newDeps(t, store), statusCreated,
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

			_, _, err := idempotency.Run(reqCtx([]byte("request-fp")), newDeps(t, store), statusCreated,
				func(context.Context) (payload, error) { return payload{}, nil })

			require.ErrorIs(t, err, apperror.ErrValidation)
		})

		t.Run("業務処理失敗時は complete せずエラーを伝播する(キー解放)", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_idempotency.NewMockStore(ctrl)
			bizErr := apperror.ErrInvalidArgument

			store.EXPECT().Claim(gomock.Any(), gomock.Any()).Return(true, nil)

			_, _, err := idempotency.Run(reqCtx([]byte("fp")), newDeps(t, store), statusCreated,
				func(context.Context) (payload, error) { return payload{}, bizErr })

			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("業務処理失敗でも claim 成立済みなら IncMiss を計上する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_idempotency.NewMockStore(ctrl)
			metrics := mock_idempotencyuc.NewMockMetrics(ctrl)

			store.EXPECT().Claim(gomock.Any(), gomock.Any()).Return(true, nil)
			metrics.EXPECT().IncMiss(gomock.Any(), op)

			_, _, err := idempotency.Run(reqCtx([]byte("fp")), newDepsWithMetrics(t, store, metrics), statusCreated,
				func(context.Context) (payload, error) { return payload{}, apperror.ErrInvalidArgument })

			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("業務処理成功後の結果が JSON 化不能なら 500(Internal) を返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_idempotency.NewMockStore(ctrl)

			store.EXPECT().Claim(gomock.Any(), gomock.Any()).Return(true, nil)

			_, _, err := idempotency.Run(reqCtx([]byte("fp")), newDeps(t, store), statusCreated,
				func(context.Context) (unmarshalable, error) { return unmarshalable{Fn: func() {}}, nil })

			require.ErrorIs(t, err, apperror.ErrInternal)
		})

		t.Run("ロック待ちタイムアウトは 409(Conflict) を返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_idempotency.NewMockStore(ctrl)

			store.EXPECT().Claim(gomock.Any(), gomock.Any()).Return(false, idempotencybndry.ErrLockTimeout)

			_, _, err := idempotency.Run(reqCtx([]byte("fp")), newDeps(t, store), statusCreated,
				func(context.Context) (payload, error) { return payload{}, nil })

			require.ErrorIs(t, err, apperror.ErrConflict)
		})

		t.Run("claim衝突直後に行が消えたレースは 409(Conflict) を返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_idempotency.NewMockStore(ctrl)

			store.EXPECT().Claim(gomock.Any(), gomock.Any()).Return(false, nil)
			store.EXPECT().Get(gomock.Any(), "user-1", "key-1").Return(nil, nil)

			_, _, err := idempotency.Run(reqCtx([]byte("fp")), newDeps(t, store), statusCreated,
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

			_, _, err := idempotency.Run(reqCtx(fp), newDeps(t, store), statusCreated,
				func(context.Context) (payload, error) { return payload{}, nil })

			require.ErrorIs(t, err, apperror.ErrInternal)
		})

		t.Run("Get が想定外エラーを返せばそのまま伝播する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_idempotency.NewMockStore(ctrl)
			wantErr := apperror.ErrInternal

			store.EXPECT().Claim(gomock.Any(), gomock.Any()).Return(false, nil)
			store.EXPECT().Get(gomock.Any(), "user-1", "key-1").Return(nil, wantErr)

			_, _, err := idempotency.Run(reqCtx([]byte("fp")), newDeps(t, store), statusCreated,
				func(context.Context) (payload, error) { return payload{}, nil })

			require.ErrorIs(t, err, wantErr)
		})

		t.Run("Claim が想定外エラーを返せばそのまま伝播する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_idempotency.NewMockStore(ctrl)
			wantErr := apperror.ErrInternal

			store.EXPECT().Claim(gomock.Any(), gomock.Any()).Return(false, wantErr)

			_, _, err := idempotency.Run(reqCtx([]byte("fp")), newDeps(t, store), statusCreated,
				func(context.Context) (payload, error) { return payload{}, nil })

			require.ErrorIs(t, err, apperror.ErrInternal)
		})

		t.Run("claim衝突直後に行が消えたレースは IncConflict を計上する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_idempotency.NewMockStore(ctrl)
			metrics := mock_idempotencyuc.NewMockMetrics(ctrl)

			store.EXPECT().Claim(gomock.Any(), gomock.Any()).Return(false, nil)
			store.EXPECT().Get(gomock.Any(), "user-1", "key-1").Return(nil, nil)
			metrics.EXPECT().IncConflict(gomock.Any(), op)

			_, _, err := idempotency.Run(reqCtx([]byte("fp")), newDepsWithMetrics(t, store, metrics), statusCreated,
				func(context.Context) (payload, error) { return payload{}, nil })

			require.ErrorIs(t, err, apperror.ErrConflict)
		})

		t.Run("Claim の想定外エラーは IncClaimFailure を計上する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_idempotency.NewMockStore(ctrl)
			metrics := mock_idempotencyuc.NewMockMetrics(ctrl)

			store.EXPECT().Claim(gomock.Any(), gomock.Any()).Return(false, apperror.ErrInternal)
			metrics.EXPECT().IncClaimFailure(gomock.Any(), op)

			_, _, err := idempotency.Run(reqCtx([]byte("fp")), newDepsWithMetrics(t, store, metrics), statusCreated,
				func(context.Context) (payload, error) { return payload{}, nil })

			require.ErrorIs(t, err, apperror.ErrInternal)
		})

		t.Run("Complete の失敗は IncCompleteFailure を計上する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_idempotency.NewMockStore(ctrl)
			metrics := mock_idempotencyuc.NewMockMetrics(ctrl)

			store.EXPECT().Claim(gomock.Any(), gomock.Any()).Return(true, nil)
			store.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(apperror.ErrInternal)
			metrics.EXPECT().IncMiss(gomock.Any(), op)
			metrics.EXPECT().IncCompleteFailure(gomock.Any(), op)

			_, _, err := idempotency.Run(reqCtx([]byte("fp")), newDepsWithMetrics(t, store, metrics), statusCreated,
				func(context.Context) (payload, error) { return payload{V: "created"}, nil })

			require.ErrorIs(t, err, apperror.ErrInternal)
		})

		t.Run("Metrics 未設定でも Complete 失敗はエラーを伝播する(計上は no-op)", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_idempotency.NewMockStore(ctrl)

			store.EXPECT().Claim(gomock.Any(), gomock.Any()).Return(true, nil)
			store.EXPECT().Complete(gomock.Any(), gomock.Any()).Return(apperror.ErrInternal)

			_, _, err := idempotency.Run(reqCtx([]byte("fp")), newDeps(t, store), statusCreated,
				func(context.Context) (payload, error) { return payload{V: "created"}, nil })

			require.ErrorIs(t, err, apperror.ErrInternal)
		})

		t.Run("claimed への並行再送は IncConflict を計上する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_idempotency.NewMockStore(ctrl)
			metrics := mock_idempotencyuc.NewMockMetrics(ctrl)
			fp := []byte("fp")

			store.EXPECT().Claim(gomock.Any(), gomock.Any()).Return(false, nil)
			store.EXPECT().Get(gomock.Any(), "user-1", "key-1").Return(&idempotencybndry.Record{
				Status:      idempotencybndry.StatusClaimed,
				Fingerprint: fp,
			}, nil)
			metrics.EXPECT().IncConflict(gomock.Any(), op)

			_, _, err := idempotency.Run(reqCtx(fp), newDepsWithMetrics(t, store, metrics), statusCreated,
				func(context.Context) (payload, error) { return payload{}, nil })

			require.ErrorIs(t, err, apperror.ErrConflict)
		})

		t.Run("ロック待ちタイムアウトは IncConflict を計上する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_idempotency.NewMockStore(ctrl)
			metrics := mock_idempotencyuc.NewMockMetrics(ctrl)

			store.EXPECT().Claim(gomock.Any(), gomock.Any()).Return(false, idempotencybndry.ErrLockTimeout)
			metrics.EXPECT().IncConflict(gomock.Any(), op)

			_, _, err := idempotency.Run(reqCtx([]byte("fp")), newDepsWithMetrics(t, store, metrics), statusCreated,
				func(context.Context) (payload, error) { return payload{}, nil })

			require.ErrorIs(t, err, apperror.ErrConflict)
		})

		t.Run("指紋不一致は IncFingerprintMismatch を計上する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			store := mock_idempotency.NewMockStore(ctrl)
			metrics := mock_idempotencyuc.NewMockMetrics(ctrl)

			store.EXPECT().Claim(gomock.Any(), gomock.Any()).Return(false, nil)
			store.EXPECT().Get(gomock.Any(), "user-1", "key-1").Return(&idempotencybndry.Record{
				Status:      idempotencybndry.StatusCompleted,
				Fingerprint: []byte("stored-fp"),
			}, nil)
			metrics.EXPECT().IncFingerprintMismatch(gomock.Any(), op)

			_, _, err := idempotency.Run(reqCtx([]byte("request-fp")), newDepsWithMetrics(t, store, metrics), statusCreated,
				func(context.Context) (payload, error) { return payload{}, nil })

			require.ErrorIs(t, err, apperror.ErrValidation)
		})
	})
}
