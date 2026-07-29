package idempotency

import (
	"context"
	"testing"

	"go-boilerplate/internal/apperror"
	idempotencybndry "go-boilerplate/internal/usecase/boundary/idempotency"
	mock_idempotency "go-boilerplate/internal/usecase/boundary/idempotency/mock"
	mock_idempotencyuc "go-boilerplate/internal/usecase/idempotency/mock"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func Test_nopMetrics_IncHit(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Metrics 未配線時の no-op 実装を panic せず呼べる", func(t *testing.T) {
			t.Parallel()

			// Metrics 未配線 → dispatcher が返す no-op 実装を呼べることを固定する（メトリクス任意＝nil-safety の契約）。
			assert.NotPanics(t, func() { Deps{}.metrics().IncHit(context.Background(), "op") })
		})
	})
}

func Test_nopMetrics_IncMiss(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Metrics 未配線時の no-op 実装を panic せず呼べる", func(t *testing.T) {
			t.Parallel()

			assert.NotPanics(t, func() { Deps{}.metrics().IncMiss(context.Background(), "op") })
		})
	})
}

func Test_nopMetrics_IncConflict(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Metrics 未配線時の no-op 実装を panic せず呼べる", func(t *testing.T) {
			t.Parallel()

			assert.NotPanics(t, func() { Deps{}.metrics().IncConflict(context.Background(), "op") })
		})
	})
}

func Test_nopMetrics_IncFingerprintMismatch(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Metrics 未配線時の no-op 実装を panic せず呼べる", func(t *testing.T) {
			t.Parallel()

			assert.NotPanics(t, func() { Deps{}.metrics().IncFingerprintMismatch(context.Background(), "op") })
		})
	})
}

func Test_nopMetrics_IncClaimFailure(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Metrics 未配線時の no-op 実装を panic せず呼べる", func(t *testing.T) {
			t.Parallel()

			assert.NotPanics(t, func() { Deps{}.metrics().IncClaimFailure(context.Background(), "op") })
		})
	})
}

func Test_nopMetrics_IncCompleteFailure(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Metrics 未配線時の no-op 実装を panic せず呼べる", func(t *testing.T) {
			t.Parallel()

			assert.NotPanics(t, func() { Deps{}.metrics().IncCompleteFailure(context.Background(), "op") })
		})
	})
}

func Test_Deps_metrics(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Metrics が nil の場合、no-op 実装を返す", func(t *testing.T) {
			t.Parallel()
			d := Deps{}

			assert.Equal(t, nopMetrics{}, d.metrics())
		})

		t.Run("Metrics が設定済みの場合、その実装をそのまま返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			impl := mock_idempotencyuc.NewMockMetrics(ctrl)
			d := Deps{Metrics: impl}

			assert.Same(t, impl, d.metrics())
		})
	})
}

func Test_decideExisting(t *testing.T) {
	t.Parallel()

	type payloadT struct {
		Value string
	}
	fp := []byte("fingerprint")
	req := Request{Scope: "user-1", Key: "key-1", Fingerprint: fp}

	newDeps := func(t *testing.T, rec *idempotencybndry.Record, getErr error) Deps {
		t.Helper()
		store := mock_idempotency.NewMockStore(gomock.NewController(t))
		store.EXPECT().Get(gomock.Any(), req.Scope, req.Key).Return(rec, getErr)
		return Deps{Store: store}
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("completed かつ fingerprint 一致なら保存済み DTO を復元して replay する", func(t *testing.T) {
			t.Parallel()

			deps := newDeps(t, &idempotencybndry.Record{
				Status:          idempotencybndry.StatusCompleted,
				Fingerprint:     fp,
				ResponsePayload: []byte(`{"Value":"stored"}`),
			}, nil)

			result, replayed, err := decideExisting[payloadT](context.Background(), deps, req)

			require.NoError(t, err)
			assert.True(t, replayed)
			assert.Equal(t, payloadT{Value: "stored"}, result)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Store.Get がエラーならそのまま伝播する", func(t *testing.T) {
			t.Parallel()

			wantErr := xerrors.New("store down")
			deps := newDeps(t, nil, wantErr)

			_, replayed, err := decideExisting[payloadT](context.Background(), deps, req)

			require.ErrorIs(t, err, wantErr)
			assert.False(t, replayed)
		})

		t.Run("claim 直後にエントリが消えたレースは ErrConflict を返す", func(t *testing.T) {
			t.Parallel()

			deps := newDeps(t, nil, nil)

			_, replayed, err := decideExisting[payloadT](context.Background(), deps, req)

			require.ErrorIs(t, err, apperror.ErrConflict)
			assert.False(t, replayed)
		})

		t.Run("fingerprint 不一致は ErrValidation を返す", func(t *testing.T) {
			t.Parallel()

			deps := newDeps(t, &idempotencybndry.Record{
				Status:      idempotencybndry.StatusCompleted,
				Fingerprint: []byte("different"),
			}, nil)

			_, replayed, err := decideExisting[payloadT](context.Background(), deps, req)

			require.ErrorIs(t, err, apperror.ErrValidation)
			assert.False(t, replayed)
		})

		t.Run("未完了ステータスは ErrConflict を返す", func(t *testing.T) {
			t.Parallel()

			deps := newDeps(t, &idempotencybndry.Record{
				Status:      idempotencybndry.StatusClaimed,
				Fingerprint: fp,
			}, nil)

			_, replayed, err := decideExisting[payloadT](context.Background(), deps, req)

			require.ErrorIs(t, err, apperror.ErrConflict)
			assert.False(t, replayed)
		})

		t.Run("completed だが保存ペイロードが不正 JSON なら ErrInternal を返す", func(t *testing.T) {
			t.Parallel()

			deps := newDeps(t, &idempotencybndry.Record{
				Status:          idempotencybndry.StatusCompleted,
				Fingerprint:     fp,
				ResponsePayload: []byte(`not-json`),
			}, nil)

			_, replayed, err := decideExisting[payloadT](context.Background(), deps, req)

			require.ErrorIs(t, err, apperror.ErrInternal)
			assert.False(t, replayed)
		})
	})
}
