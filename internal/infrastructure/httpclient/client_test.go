package httpclient_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/infrastructure/httpclient"
	"go-boilerplate/internal/infrastructure/system"
	"go-boilerplate/internal/observability"
	mock_clock "go-boilerplate/internal/usecase/boundary/clock/mock"
	clocktestkit "go-boilerplate/internal/usecase/boundary/clock/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// countingServer は、リクエスト回数を数えつつ固定ステータスを返すテストサーバを生成します。
func countingServer(t *testing.T, status int) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

// retryProfile は、retry 検証用に最大 3 試行へ固定した Profile の Registry を返します。
func retryProfile() httpclient.Registry {
	p := httpclient.DefaultProfile()
	p.MaxAttempts = 3
	p.BaseBackoff = time.Millisecond
	p.MaxBackoff = 2 * time.Millisecond
	return httpclient.NewRegistry(map[httpclient.Downstream]httpclient.Profile{"retry": p})
}

// newClient は、テスト用の substrate を生成します（plain transport / noop sleeper / noop metrics）。
func newClient(t *testing.T, registry httpclient.Registry) httpclient.Client {
	t.Helper()
	return httpclient.New(
		http.DefaultTransport,
		clocktestkit.NewNoopSleeper(t),
		registry,
		observability.NewNoopHTTPClientMetrics(t),
	)
}

func TestClientDo(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("2xxはレスポンスとnilを返しボディを読み切る", func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("X-Trace", "ok")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"ok":true}`))
			}))
			t.Cleanup(srv.Close)

			client := newClient(t, httpclient.NewRegistry(nil))
			resp, err := client.Do(context.Background(), &httpclient.Request{
				Downstream: "sample",
				Method:     httpclient.MethodGet,
				URL:        srv.URL,
			})

			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, []byte(`{"ok":true}`), resp.Body)
			assert.Equal(t, []string{"ok"}, resp.Header["X-Trace"])
		})

		t.Run("POSTのボディと冪等性キーがサーバへ伝搬する", func(t *testing.T) {
			t.Parallel()

			var gotBody []byte
			var gotKey string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotBody, _ = io.ReadAll(r.Body)
				gotKey = r.Header.Get("Idempotency-Key")
				w.WriteHeader(http.StatusOK)
			}))
			t.Cleanup(srv.Close)

			client := newClient(t, httpclient.NewRegistry(nil))
			_, err := client.Do(context.Background(), &httpclient.Request{
				Downstream:     "sample",
				Method:         httpclient.MethodPost,
				URL:            srv.URL,
				Body:           []byte(`{"v":1}`),
				IdempotencyKey: "key-123",
			})

			require.NoError(t, err)
			assert.Equal(t, []byte(`{"v":1}`), gotBody)
			assert.Equal(t, "key-123", gotKey)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		statusCases := map[string]struct {
			status int
			want   error
		}{
			"404はErrNotFound":        {status: http.StatusNotFound, want: apperror.ErrNotFound},
			"400はErrInvalidArgument": {status: http.StatusBadRequest, want: apperror.ErrInvalidArgument},
			"429はErrTooManyRequests": {status: http.StatusTooManyRequests, want: apperror.ErrTooManyRequests},
			"500はErrUnavailable":     {status: http.StatusInternalServerError, want: apperror.ErrUnavailable},
		}

		for name, tc := range statusCases {
			t.Run(name+"_でもレスポンスは返す", func(t *testing.T) {
				t.Parallel()

				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(tc.status)
					_, _ = w.Write([]byte("error-body"))
				}))
				t.Cleanup(srv.Close)

				client := newClient(t, httpclient.NewRegistry(nil))
				resp, err := client.Do(context.Background(), &httpclient.Request{
					Downstream: "sample",
					Method:     httpclient.MethodGet,
					URL:        srv.URL,
				})

				require.ErrorIs(t, err, tc.want)
				require.NotNil(t, resp)
				assert.Equal(t, tc.status, resp.StatusCode)
				assert.Equal(t, []byte("error-body"), resp.Body)
			})
		}

		t.Run("transport失敗はErrUnavailableを返しレスポンスはnil", func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
			url := srv.URL
			srv.Close() // 先に閉じて接続不能にする

			client := newClient(t, httpclient.NewRegistry(nil))
			resp, err := client.Do(context.Background(), &httpclient.Request{
				Downstream: "sample",
				Method:     httpclient.MethodGet,
				URL:        url,
			})

			require.ErrorIs(t, err, apperror.ErrUnavailable)
			assert.Nil(t, resp)
		})

		t.Run("ctxキャンセル済みはErrCanceledを返す", func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			t.Cleanup(srv.Close)

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			client := newClient(t, httpclient.NewRegistry(nil))
			resp, err := client.Do(ctx, &httpclient.Request{
				Downstream: "sample",
				Method:     httpclient.MethodGet,
				URL:        srv.URL,
			})

			require.ErrorIs(t, err, apperror.ErrCanceled)
			assert.Nil(t, resp)
		})

		t.Run("不正なURLはErrInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()

			client := newClient(t, httpclient.NewRegistry(nil))
			resp, err := client.Do(context.Background(), &httpclient.Request{
				Downstream: "sample",
				Method:     httpclient.MethodGet,
				URL:        "://invalid",
			})

			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
			assert.Nil(t, resp)
		})

		t.Run("ボディが上限超過ならErrUnavailableを返す", func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("0123456789"))
			}))
			t.Cleanup(srv.Close)

			profile := httpclient.DefaultProfile()
			profile.MaxResponseBytes = 4
			registry := httpclient.NewRegistry(map[httpclient.Downstream]httpclient.Profile{"tiny": profile})

			client := newClient(t, registry)
			resp, err := client.Do(context.Background(), &httpclient.Request{
				Downstream: "tiny",
				Method:     httpclient.MethodGet,
				URL:        srv.URL,
			})

			require.ErrorIs(t, err, apperror.ErrUnavailable)
			assert.Nil(t, resp)
		})
	})
}

func TestNewRegistry(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("登録済みキーは対応Profileを返す", func(t *testing.T) {
			t.Parallel()

			want := httpclient.DefaultProfile()
			want.MaxAttempts = 99
			registry := httpclient.NewRegistry(map[httpclient.Downstream]httpclient.Profile{"a": want})

			assert.Equal(t, 99, registry.Profile("a").MaxAttempts)
		})

		t.Run("未登録キーはDefaultProfileにfallbackする", func(t *testing.T) {
			t.Parallel()

			registry := httpclient.NewRegistry(nil)
			assert.Equal(t, httpclient.DefaultProfile(), registry.Profile("unknown"))
		})
	})
}

func TestClientDoMinimumAttempt(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("MaxAttemptsが0でも最低1回試行しnil_nilを返さない", func(t *testing.T) {
			t.Parallel()

			srv, hits := countingServer(t, http.StatusOK)

			profile := httpclient.DefaultProfile()
			profile.MaxAttempts = 0
			registry := httpclient.NewRegistry(map[httpclient.Downstream]httpclient.Profile{"zero": profile})

			client := newClient(t, registry)
			resp, err := client.Do(context.Background(), &httpclient.Request{
				Downstream: "zero",
				Method:     httpclient.MethodGet,
				URL:        srv.URL,
			})

			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, int32(1), hits.Load())
		})
	})
}

func TestClientDoRetry(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("503はMaxAttemptsまでリトライする", func(t *testing.T) {
			t.Parallel()

			srv, hits := countingServer(t, http.StatusServiceUnavailable)
			client := newClient(t, retryProfile())

			_, err := client.Do(context.Background(), &httpclient.Request{
				Downstream: "retry",
				Method:     httpclient.MethodGet,
				URL:        srv.URL,
			})

			require.ErrorIs(t, err, apperror.ErrUnavailable)
			assert.Equal(t, int32(3), hits.Load())
		})

		t.Run("4xxはリトライしない", func(t *testing.T) {
			t.Parallel()

			srv, hits := countingServer(t, http.StatusBadRequest)
			client := newClient(t, retryProfile())

			_, err := client.Do(context.Background(), &httpclient.Request{
				Downstream: "retry",
				Method:     httpclient.MethodGet,
				URL:        srv.URL,
			})

			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
			assert.Equal(t, int32(1), hits.Load())
		})

		t.Run("AllowRetryなしのPOSTは503でも1回しか飛ばない", func(t *testing.T) {
			t.Parallel()

			srv, hits := countingServer(t, http.StatusServiceUnavailable)
			client := newClient(t, retryProfile())

			_, err := client.Do(context.Background(), &httpclient.Request{
				Downstream: "retry",
				Method:     httpclient.MethodPost,
				URL:        srv.URL,
			})

			require.ErrorIs(t, err, apperror.ErrUnavailable)
			assert.Equal(t, int32(1), hits.Load())
		})

		t.Run("AllowRetryと冪等性キーありのPOSTはリトライする", func(t *testing.T) {
			t.Parallel()

			srv, hits := countingServer(t, http.StatusServiceUnavailable)
			client := newClient(t, retryProfile())

			_, err := client.Do(context.Background(), &httpclient.Request{
				Downstream:     "retry",
				Method:         httpclient.MethodPost,
				URL:            srv.URL,
				IdempotencyKey: "key-1",
				AllowRetry:     true,
			})

			require.ErrorIs(t, err, apperror.ErrUnavailable)
			assert.Equal(t, int32(3), hits.Load())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("AllowRetryありで冪等性キーなしはErrInvalidArgumentを返し送信しない", func(t *testing.T) {
			t.Parallel()

			srv, hits := countingServer(t, http.StatusOK)
			client := newClient(t, retryProfile())

			resp, err := client.Do(context.Background(), &httpclient.Request{
				Downstream: "retry",
				Method:     httpclient.MethodPost,
				URL:        srv.URL,
				AllowRetry: true,
			})

			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
			assert.Nil(t, resp)
			assert.Equal(t, int32(0), hits.Load())
		})

		t.Run("backoffのスリープ中にctxがキャンセルされたらErrCanceledを返す", func(t *testing.T) {
			t.Parallel()

			srv, _ := countingServer(t, http.StatusServiceUnavailable)

			ctrl := gomock.NewController(t)
			sleeper := mock_clock.NewMockSleeper(ctrl)
			sleeper.EXPECT().Sleep(gomock.Any(), gomock.Any()).Return(context.Canceled).AnyTimes()

			client := httpclient.New(http.DefaultTransport, sleeper, retryProfile(), observability.NewNoopHTTPClientMetrics(t))

			_, err := client.Do(context.Background(), &httpclient.Request{
				Downstream: "retry",
				Method:     httpclient.MethodGet,
				URL:        srv.URL,
			})

			require.ErrorIs(t, err, apperror.ErrCanceled)
		})
	})
}

func TestClientDoBackoff(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("リトライごとのbackoffが指数cap内のfull_jitter範囲に収まる", func(t *testing.T) {
			t.Parallel()

			srv, _ := countingServer(t, http.StatusServiceUnavailable)

			profile := httpclient.DefaultProfile()
			profile.MaxAttempts = 4
			profile.BaseBackoff = 10 * time.Millisecond
			profile.MaxBackoff = time.Second
			registry := httpclient.NewRegistry(map[httpclient.Downstream]httpclient.Profile{"retry": profile})

			var slept []time.Duration
			ctrl := gomock.NewController(t)
			sleeper := mock_clock.NewMockSleeper(ctrl)
			sleeper.EXPECT().Sleep(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, d time.Duration) error {
					slept = append(slept, d)
					return nil
				}).AnyTimes()

			client := httpclient.New(http.DefaultTransport, sleeper, registry, observability.NewNoopHTTPClientMetrics(t))
			_, err := client.Do(context.Background(), &httpclient.Request{
				Downstream: "retry",
				Method:     httpclient.MethodGet,
				URL:        srv.URL,
			})

			require.ErrorIs(t, err, apperror.ErrUnavailable)
			require.Len(t, slept, 3) // MaxAttempts-1 回のリトライ
			for i, d := range slept {
				upperBound := min(profile.BaseBackoff<<i, profile.MaxBackoff)
				assert.GreaterOrEqual(t, d, time.Duration(0))
				assert.LessOrEqual(t, d, upperBound)
			}
		})
	})
}

func TestClientDoDeadline(t *testing.T) {
	t.Parallel()

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("overallデッドラインを超えるリトライは打ち切られMaxAttemptsに達しない", func(t *testing.T) {
			t.Parallel()

			srv, hits := countingServer(t, http.StatusServiceUnavailable)

			profile := httpclient.DefaultProfile()
			profile.MaxAttempts = 20
			profile.BaseBackoff = 20 * time.Millisecond
			profile.MaxBackoff = 20 * time.Millisecond
			profile.OverallTimeout = 60 * time.Millisecond
			registry := httpclient.NewRegistry(map[httpclient.Downstream]httpclient.Profile{"retry": profile})

			// 実時間を消費する sleeper を使い、overall デッドラインで打ち切られることを検証する。
			client := httpclient.New(http.DefaultTransport, system.NewSleeper(), registry, observability.NewNoopHTTPClientMetrics(t))
			_, err := client.Do(context.Background(), &httpclient.Request{
				Downstream: "retry",
				Method:     httpclient.MethodGet,
				URL:        srv.URL,
			})

			require.ErrorIs(t, err, apperror.ErrUnavailable)
			assert.Less(t, hits.Load(), int32(20))
			assert.GreaterOrEqual(t, hits.Load(), int32(1))
		})
	})
}

func TestClientDoBreaker(t *testing.T) {
	t.Parallel()

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("連続失敗でbreakerがopenになり以降は即fail-fastする", func(t *testing.T) {
			t.Parallel()

			srv, hits := countingServer(t, http.StatusServiceUnavailable)

			profile := httpclient.DefaultProfile()
			profile.MaxAttempts = 1 // retry を無効化し breaker の挙動を単純化
			profile.Breaker = httpclient.BreakerConfig{
				FailureThreshold: 0.5,
				MinRequests:      2,
				OpenDuration:     time.Hour, // テスト中は half-open へ遷移させない
				HalfOpenProbes:   1,
			}
			registry := httpclient.NewRegistry(map[httpclient.Downstream]httpclient.Profile{"brk": profile})
			client := newClient(t, registry)

			req := &httpclient.Request{Downstream: "brk", Method: httpclient.MethodGet, URL: srv.URL}

			// MinRequests(2) 件の失敗で open する。
			_, err1 := client.Do(context.Background(), req)
			require.ErrorIs(t, err1, apperror.ErrUnavailable)
			_, err2 := client.Do(context.Background(), req)
			require.ErrorIs(t, err2, apperror.ErrUnavailable)

			hitsBeforeFastFail := hits.Load()

			// open 後はサーバへ到達せず即 fail-fast する。
			_, err3 := client.Do(context.Background(), req)
			require.ErrorIs(t, err3, apperror.ErrUnavailable)
			assert.Equal(t, hitsBeforeFastFail, hits.Load())
		})
	})
}

func TestClientDoBudget(t *testing.T) {
	t.Parallel()

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("retry_budget枯渇後はリトライせず初回試行のみになる", func(t *testing.T) {
			t.Parallel()

			srv, hits := countingServer(t, http.StatusServiceUnavailable)

			profile := httpclient.DefaultProfile()
			profile.MaxAttempts = 2 // 1 リクエストあたり retry 1 回(=1トークン消費)
			profile.BaseBackoff = time.Millisecond
			profile.MaxBackoff = time.Millisecond
			profile.RetryBudgetRatio = 0          // 補充しないため枯渇させやすい
			profile.Breaker.MinRequests = 1 << 30 // breaker は開かせない
			registry := httpclient.NewRegistry(map[httpclient.Downstream]httpclient.Profile{"bdg": profile})
			client := newClient(t, registry)

			req := &httpclient.Request{Downstream: "bdg", Method: httpclient.MethodGet, URL: srv.URL}

			// 初期トークン(=10)を消費し切るまでリクエストを繰り返す。
			for range 10 {
				_, err := client.Do(context.Background(), req)
				require.ErrorIs(t, err, apperror.ErrUnavailable)
			}

			// 枯渇後の 1 リクエストは初回試行のみ（retry なし）。
			before := hits.Load()
			_, err := client.Do(context.Background(), req)
			require.ErrorIs(t, err, apperror.ErrUnavailable)
			assert.Equal(t, int32(1), hits.Load()-before)
		})
	})
}

func TestClientDoRetryAfter(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Retry-Afterヘッダがあればバックオフより優先して待機する", func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Retry-After", "2")
				w.WriteHeader(http.StatusServiceUnavailable)
			}))
			t.Cleanup(srv.Close)

			profile := httpclient.DefaultProfile()
			profile.MaxAttempts = 2
			profile.BaseBackoff = 10 * time.Millisecond
			profile.MaxBackoff = 20 * time.Millisecond // バックオフ上限より Retry-After(2s) が大きい
			profile.OverallTimeout = time.Hour
			registry := httpclient.NewRegistry(map[httpclient.Downstream]httpclient.Profile{"ra": profile})

			var slept []time.Duration
			ctrl := gomock.NewController(t)
			sleeper := mock_clock.NewMockSleeper(ctrl)
			sleeper.EXPECT().Sleep(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, d time.Duration) error {
					slept = append(slept, d)
					return nil
				}).AnyTimes()

			client := httpclient.New(http.DefaultTransport, sleeper, registry, observability.NewNoopHTTPClientMetrics(t))
			_, err := client.Do(context.Background(), &httpclient.Request{
				Downstream: "ra",
				Method:     httpclient.MethodGet,
				URL:        srv.URL,
			})

			require.ErrorIs(t, err, apperror.ErrUnavailable)
			require.Len(t, slept, 1)
			assert.Equal(t, 2*time.Second, slept[0]) // バックオフ(<=20ms)ではなく Retry-After(2s)
		})
	})
}

func TestClientDoTimeout(t *testing.T) {
	t.Parallel()

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("per-attemptタイムアウト超過はErrUnavailableを返す", func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				time.Sleep(200 * time.Millisecond)
				w.WriteHeader(http.StatusOK)
			}))
			t.Cleanup(srv.Close)

			profile := httpclient.DefaultProfile()
			profile.PerAttemptTimeout = 20 * time.Millisecond
			profile.OverallTimeout = 50 * time.Millisecond
			registry := httpclient.NewRegistry(map[httpclient.Downstream]httpclient.Profile{"slow": profile})

			client := newClient(t, registry)
			resp, err := client.Do(context.Background(), &httpclient.Request{
				Downstream: "slow",
				Method:     httpclient.MethodGet,
				URL:        srv.URL,
			})

			require.ErrorIs(t, err, apperror.ErrUnavailable)
			assert.Nil(t, resp)
		})
	})
}
