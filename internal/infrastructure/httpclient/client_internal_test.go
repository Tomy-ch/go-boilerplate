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

	const ds Downstream = "acct"

	newTestClient := func(t *testing.T) *client {
		t.Helper()
		clk := clocktestkit.NewStepClock(time.Now(), 0)
		c, ok := New(
			observability.NewNoopHTTPClientTransport(t),
			clk,
			clk,
			NewRegistry(map[Downstream]Profile{ds: DefaultProfile()}),
			observability.NewNoopHTTPClientMetrics(t),
		).(*client)
		require.True(t, ok)
		return c
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("2xx はステータス・ヘッダ・ボディを詰めた Response を返す", func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("X-Kind", "ok")
				_, _ = w.Write([]byte("body"))
			}))
			t.Cleanup(srv.Close)

			resp, err := newTestClient(t).attempt(context.Background(), NewRequest(MethodGet(), ds, srv.URL), DefaultProfile())

			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, []string{"ok"}, resp.Header["X-Kind"])
			assert.Equal(t, []byte("body"), resp.Body)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("URL が不正な場合は送信せず ErrInvalidArgument を返す", func(t *testing.T) {
			t.Parallel()

			resp, err := newTestClient(t).attempt(context.Background(), NewRequest(MethodGet(), ds, "://invalid"), DefaultProfile())

			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
			assert.Nil(t, resp)
		})

		t.Run("接続に失敗した場合は transport エラーとして正規化する", func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
			url := srv.URL
			srv.Close() // 接続先を落として transport 失敗を確定させる

			resp, err := newTestClient(t).attempt(context.Background(), NewRequest(MethodGet(), ds, url), DefaultProfile())

			require.ErrorIs(t, err, apperror.ErrUnavailable)
			assert.Nil(t, resp)
		})

		t.Run("ボディが上限を超える場合は errResponseTooLarge をそのまま返す", func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("too long body"))
			}))
			t.Cleanup(srv.Close)

			profile := DefaultProfile()
			profile.MaxResponseBytes = 1

			resp, err := newTestClient(t).attempt(context.Background(), NewRequest(MethodGet(), ds, srv.URL), profile)

			require.ErrorIs(t, err, errResponseTooLarge)
			assert.Nil(t, resp)
		})

		t.Run("ボディの読み取り自体が失敗した場合は transport エラーとして正規化する", func(t *testing.T) {
			t.Parallel()

			// Content-Length を過大申告したまま接続を切り、上限超過ではない純粋な read 失敗を起こす。
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				hj, ok := w.(http.Hijacker)
				if !ok {
					return
				}
				conn, buf, err := hj.Hijack()
				if err != nil {
					return
				}
				_, _ = buf.WriteString("HTTP/1.1 200 OK\r\nContent-Length: 100\r\n\r\nhello")
				_ = buf.Flush()
				_ = conn.Close()
			}))
			t.Cleanup(srv.Close)

			resp, err := newTestClient(t).attempt(context.Background(), NewRequest(MethodGet(), ds, srv.URL), DefaultProfile())

			require.ErrorIs(t, err, apperror.ErrUnavailable)
			require.NotErrorIs(t, err, errResponseTooLarge) // 上限超過ではなく read 失敗として扱う
			assert.Nil(t, resp)
		})

		t.Run("エラーステータスは Response を保ったまま対応するアプリエラーを返す", func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte("down"))
			}))
			t.Cleanup(srv.Close)

			resp, err := newTestClient(t).attempt(context.Background(), NewRequest(MethodGet(), ds, srv.URL), DefaultProfile())

			require.ErrorIs(t, err, apperror.ErrUnavailable)
			require.NotNil(t, resp) // 呼び出し元が status / body を見られるよう Response は返す
			assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
			assert.Equal(t, []byte("down"), resp.Body)
		})
	})
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

	// newRecordingClient は、計上内容を読み出せる metrics を背にした client を返す。
	newRecordingClient := func(t *testing.T) (*client, *observability.ObservedHTTPClientMetrics) {
		t.Helper()
		obs := observability.NewObservedHTTPClientMetrics(t)
		clk := clocktestkit.NewStepClock(time.Now(), 0)
		c, ok := New(
			observability.NewNoopHTTPClientTransport(t),
			clk,
			clk,
			NewRegistry(nil),
			obs.HTTPClientMetrics,
		).(*client)
		require.True(t, ok)
		return c, obs
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("成功応答は status_class 付きで requests のみ計上する", func(t *testing.T) {
			t.Parallel()

			c, obs := newRecordingClient(t)

			c.recordOutcome(context.Background(), "acct", &Response{StatusCode: http.StatusOK}, nil)

			assert.Equal(t, []string{"2xx"}, obs.LabelValues(t, "httpclient.requests", "status_class"))
			assert.Empty(t, obs.LabelValues(t, "httpclient.errors", "reason"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("応答ありのエラーは http_ + status_class を reason に計上する", func(t *testing.T) {
			t.Parallel()

			c, obs := newRecordingClient(t)

			c.recordOutcome(
				context.Background(), "acct", &Response{StatusCode: http.StatusServiceUnavailable}, apperror.ErrUnavailable)

			assert.Equal(t, []string{"http_5xx"}, obs.LabelValues(t, "httpclient.errors", "reason"))
		})

		t.Run("サーキット開放は circuit_open を reason に計上する", func(t *testing.T) {
			t.Parallel()

			c, obs := newRecordingClient(t)

			c.recordOutcome(context.Background(), "acct", nil, errCircuitOpen)

			assert.Equal(t, []string{"circuit_open"}, obs.LabelValues(t, "httpclient.errors", "reason"))
			// 応答が無いので requests は計上しない。
			assert.Empty(t, obs.LabelValues(t, "httpclient.requests", "status_class"))
		})

		t.Run("キャンセルは canceled を reason に計上する", func(t *testing.T) {
			t.Parallel()

			c, obs := newRecordingClient(t)

			c.recordOutcome(context.Background(), "acct", nil, apperror.ErrCanceled)

			assert.Equal(t, []string{"canceled"}, obs.LabelValues(t, "httpclient.errors", "reason"))
		})

		t.Run("分類できないエラーは transport を reason に計上する", func(t *testing.T) {
			t.Parallel()

			c, obs := newRecordingClient(t)

			c.recordOutcome(context.Background(), "acct", nil, xerrors.New("dial failed"))

			assert.Equal(t, []string{"transport"}, obs.LabelValues(t, "httpclient.errors", "reason"))
		})
	})
}

func Test_client_canRetryWithin(t *testing.T) {
	t.Parallel()

	base := time.Unix(1000, 0)

	// newFixedClockClient は、時刻が進まない clock を持つ client を返す。
	newFixedClockClient := func() *client {
		return &client{clk: clocktestkit.NewStepClock(base, 0)}
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("backoff待機後も期限より前なら次の試行を許可する", func(t *testing.T) {
			t.Parallel()

			assert.True(t, newFixedClockClient().canRetryWithin(base.Add(time.Second), 999*time.Millisecond))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("backoff待機後が期限ちょうどなら次の試行を許可しない", func(t *testing.T) {
			t.Parallel()

			assert.False(t, newFixedClockClient().canRetryWithin(base.Add(time.Second), time.Second))
		})

		t.Run("backoff待機後が期限を超えるなら次の試行を許可しない", func(t *testing.T) {
			t.Parallel()

			assert.False(t, newFixedClockClient().canRetryWithin(base.Add(time.Second), 2*time.Second))
		})

		t.Run("既に期限を過ぎている場合は待機なしでも次の試行を許可しない", func(t *testing.T) {
			t.Parallel()

			assert.False(t, newFixedClockClient().canRetryWithin(base.Add(-time.Second), 0))
		})
	})
}

func Test_noFollowRedirect(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("リダイレクトを追従せず最終レスポンスを返す指示を返す", func(t *testing.T) {
			t.Parallel()

			req, err := http.NewRequestWithContext(
				context.Background(), http.MethodGet, "https://example.com/redirected", nil)
			require.NoError(t, err)

			require.ErrorIs(t, noFollowRedirect(req, []*http.Request{req}), http.ErrUseLastResponse)
		})
	})
}
