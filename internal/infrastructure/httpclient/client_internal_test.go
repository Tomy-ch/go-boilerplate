package httpclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/observability"
	clocktestkit "go-boilerplate/internal/usecase/boundary/clock/testkit"
	"go-boilerplate/pkg/xerrors"
)

var errRead = xerrors.New("read failed")

// errReader は、常に read エラーを返す io.Reader です。
type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errRead }

func Test_Request_validate(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("必須項目が揃っていればnilを返す", func(t *testing.T) {
			t.Parallel()

			req := &Request{downstream: "sample", method: MethodGet(), url: "http://example.com"}

			require.NoError(t, req.validate())
		})

		t.Run("AllowRetryありでも冪等性キーがあればnilを返す", func(t *testing.T) {
			t.Parallel()

			req := &Request{
				downstream:     "sample",
				method:         MethodPost(),
				url:            "http://example.com",
				allowRetry:     true,
				idempotencyKey: "key-1",
			}

			require.NoError(t, req.validate())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Downstreamが空ならerrDownstreamRequiredを返す", func(t *testing.T) {
			t.Parallel()

			req := &Request{downstream: "", method: MethodGet(), url: "http://example.com"}

			require.ErrorIs(t, req.validate(), errDownstreamRequired)
		})

		t.Run("Methodがゼロ値ならerrMethodRequiredを返す", func(t *testing.T) {
			t.Parallel()

			req := &Request{downstream: "sample", method: Method{}, url: "http://example.com"}

			require.ErrorIs(t, req.validate(), errMethodRequired)
		})

		t.Run("AllowRetryありで冪等性キーが空ならerrIdempotencyKeyRequiredを返す", func(t *testing.T) {
			t.Parallel()

			req := &Request{
				downstream: "sample",
				method:     MethodPost(),
				url:        "http://example.com",
				allowRetry: true,
			}

			require.ErrorIs(t, req.validate(), errIdempotencyKeyRequired)
		})
	})
}

func Test_readBody(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("上限以内のボディを読み切って返す", func(t *testing.T) {
			t.Parallel()

			data, err := readBody(strings.NewReader("hello"), 10)

			require.NoError(t, err)
			assert.Equal(t, []byte("hello"), data)
		})

		t.Run("上限ちょうどのボディは読み切れる", func(t *testing.T) {
			t.Parallel()

			data, err := readBody(strings.NewReader("hello"), 5)

			require.NoError(t, err)
			assert.Equal(t, []byte("hello"), data)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("上限を超えるボディはerrResponseTooLargeを返す", func(t *testing.T) {
			t.Parallel()

			data, err := readBody(strings.NewReader("hello"), 4)

			require.ErrorIs(t, err, errResponseTooLarge)
			assert.Nil(t, data)
		})

		t.Run("read中のエラーをそのまま返す", func(t *testing.T) {
			t.Parallel()

			_, err := readBody(errReader{}, 10)

			require.ErrorIs(t, err, errRead)
		})
	})
}

func Test_client_attempt(t *testing.T) {
	t.Parallel()
	t.Skip("client.attempt は client_test.go の Test_client_Do_Send（2xx/4xx/5xx/transport失敗/URL不正/ボディ上限超過）で httptest サーバ経由の統合テストとして網羅されている")
}

// Test_client_doWithRetry は、retry ループの分岐網羅（backoff / deadline / breaker / budget / minimum-attempt）を
// client_test.go の Test_client_Do_Retry/Backoff/Deadline/Breaker/Budget/MinimumAttempt が httptest サーバ経由で
// 担う一方、外部テストからは観測しにくい内部状態（client.budget.tokens）を直接固定する。ここでは deadline で
// 打ち切られる retry が budget を消費しない会計整合を pin する。budget 消費を canRetryWithin より先に行う旧実装
// では、打ち切られた分まで消費して残量が 1 少なくなる回帰を検出する。
func Test_client_doWithRetry(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("deadline で打ち切られた retry は budget を消費せず残量が実施 retry 数に一致する", func(t *testing.T) {
			t.Parallel()

			var hits atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				hits.Add(1)
				w.WriteHeader(http.StatusServiceUnavailable)
			}))
			t.Cleanup(srv.Close)

			profile := DefaultProfile()
			profile.MaxAttempts = 20
			// backoff は極小にし、打ち切りは StepClock(20ms/step) と OverallTimeout(60ms) の関係のみで決める。
			profile.BaseBackoff = time.Millisecond
			profile.MaxBackoff = time.Millisecond
			profile.OverallTimeout = 60 * time.Millisecond
			profile.RetryBudgetRatio = 100        // budget が deadline より先に絞らないよう十分に確保
			profile.Breaker.MinRequests = 1 << 30 // breaker を開かせない
			const ds Downstream = "acct"
			registry := NewRegistry(map[Downstream]Profile{ds: profile})

			// Sleep で fake 時刻を 20ms/サイクル進める clock を注入し、注入 clock 基準の
			// canRetryWithin（overallDeadline 到達）で決定的に打ち切る（実時間・jitter 非依存）。
			fakeClock := clocktestkit.NewStepClock(time.Now().Add(time.Hour), 20*time.Millisecond)
			c, ok := New(
				observability.NewNoopHTTPClientTransport(t),
				fakeClock,
				fakeClock,
				registry,
				observability.NewNoopHTTPClientMetrics(t),
			).(*client)
			require.True(t, ok)

			_, err := c.Do(context.Background(), NewRequest(MethodGet(), ds, srv.URL))
			require.ErrorIs(t, err, apperror.ErrUnavailable)

			// 20ms/step × 3 backoff で fakeNow がちょうど deadline(60ms) に達し、hits=4（= retry 3 回）。
			// 4 回目の attempt は canRetryWithin=false で打ち切られ、budget を消費しない。
			require.Equal(t, int32(4), hits.Load())

			initial := min(retryBudgetMaxTokens, retryBudgetInitialTokens+profile.RetryBudgetRatio)
			retriesPerformed := hits.Load() - 1
			// 残量 = 初期在庫 - 実施 retry 数。旧実装は打ち切り分も消費して残量が 1 少なくなる。
			assert.InDelta(t, initial-float64(retriesPerformed)*retryBudgetRetryCost, c.budget.tokens[ds], 1e-9)
		})
	})
}

func Test_client_recordOutcome(t *testing.T) {
	t.Parallel()
	t.Skip("client.recordOutcome は client_test.go の Test_client_Do_Send（http_/circuit_open/canceled/transport の各 error class）で網羅されている")
}
