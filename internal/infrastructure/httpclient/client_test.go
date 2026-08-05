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
		observability.NewNoopHTTPClientTransport(t),
		clocktestkit.NewNoopSleeper(t),
		system.NewClock(),
		registry,
		observability.NewNoopHTTPClientMetrics(t),
	)
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("依存を渡して構築すると非nilのClientを返す", func(t *testing.T) {
			t.Parallel()

			client := httpclient.New(
				observability.NewNoopHTTPClientTransport(t),
				clocktestkit.NewNoopSleeper(t),
				system.NewClock(),
				httpclient.NewRegistry(nil),
				observability.NewNoopHTTPClientMetrics(t),
			)

			assert.NotNil(t, client)
		})
	})
}

func Test_client_Do_Send(t *testing.T) {
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
			resp, err := client.Do(context.Background(), httpclient.NewRequest(httpclient.MethodGet(), "sample", srv.URL))

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
			_, err := client.Do(context.Background(), httpclient.NewRequest(
				httpclient.MethodPost(), "sample", srv.URL,
				httpclient.WithBody([]byte(`{"v":1}`)),
				httpclient.WithIdempotencyKey("key-123"),
			))

			require.NoError(t, err)
			assert.Equal(t, []byte(`{"v":1}`), gotBody)
			assert.Equal(t, "key-123", gotKey)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		// ステータスコードごとに、apperror へ正規化しつつレスポンス本体も返すことを確認する。
		assertStatusMapped := func(t *testing.T, status int, want error) {
			t.Helper()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
				_, _ = w.Write([]byte("error-body"))
			}))
			t.Cleanup(srv.Close)

			client := newClient(t, httpclient.NewRegistry(nil))
			resp, err := client.Do(context.Background(), httpclient.NewRequest(httpclient.MethodGet(), "sample", srv.URL))

			require.ErrorIs(t, err, want)
			require.NotNil(t, resp)
			assert.Equal(t, status, resp.StatusCode)
			assert.Equal(t, []byte("error-body"), resp.Body)
		}

		t.Run("404はErrNotFoundでもレスポンスは返す", func(t *testing.T) {
			t.Parallel()
			assertStatusMapped(t, http.StatusNotFound, apperror.ErrNotFound)
		})

		t.Run("400はErrInvalidArgumentでもレスポンスは返す", func(t *testing.T) {
			t.Parallel()
			assertStatusMapped(t, http.StatusBadRequest, apperror.ErrInvalidArgument)
		})

		t.Run("429はErrTooManyRequestsでもレスポンスは返す", func(t *testing.T) {
			t.Parallel()
			assertStatusMapped(t, http.StatusTooManyRequests, apperror.ErrTooManyRequests)
		})

		t.Run("500はErrUnavailableでもレスポンスは返す", func(t *testing.T) {
			t.Parallel()
			assertStatusMapped(t, http.StatusInternalServerError, apperror.ErrUnavailable)
		})

		t.Run("transport失敗はErrUnavailableを返しレスポンスはnil", func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
			url := srv.URL
			srv.Close() // 先に閉じて接続不能にする

			client := newClient(t, httpclient.NewRegistry(nil))
			resp, err := client.Do(context.Background(), httpclient.NewRequest(httpclient.MethodGet(), "sample", url))

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
			resp, err := client.Do(ctx, httpclient.NewRequest(httpclient.MethodGet(), "sample", srv.URL))

			require.ErrorIs(t, err, apperror.ErrCanceled)
			assert.Nil(t, resp)
		})

		t.Run("不正なURLはErrInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()

			client := newClient(t, httpclient.NewRegistry(nil))
			resp, err := client.Do(context.Background(), httpclient.NewRequest(httpclient.MethodGet(), "sample", "://invalid"))

			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
			assert.Nil(t, resp)
		})

		t.Run("ボディ読み取り中に接続が切れるとErrUnavailableを返す", func(t *testing.T) {
			t.Parallel()

			// Content-Length を過大申告しつつ本文を途中で切って接続を閉じ、ボディ読み取りを失敗させる。
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

			profile := httpclient.DefaultProfile()
			profile.MaxAttempts = 1 // 再送させず readBody の失敗経路を単独で踏ませる
			registry := httpclient.NewRegistry(map[httpclient.Downstream]httpclient.Profile{"body": profile})

			client := newClient(t, registry)
			resp, err := client.Do(context.Background(), httpclient.NewRequest(httpclient.MethodGet(), "body", srv.URL))

			require.ErrorIs(t, err, apperror.ErrUnavailable)
			assert.Nil(t, resp)
		})

		t.Run("ボディ上限超過はErrUnavailableを返しレスポンスはnil", func(t *testing.T) {
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
			resp, err := client.Do(context.Background(), httpclient.NewRequest(httpclient.MethodGet(), "tiny", srv.URL))

			// 下流起因の応答異常なので HTTP 語彙は transport 失敗と同じ ErrUnavailable(503)。
			// 非 retry 化は isRetryableOutcome が errResponseTooLarge を除外して担う。
			require.ErrorIs(t, err, apperror.ErrUnavailable)
			assert.Nil(t, resp)
		})
	})
}

// truncatingHijackServer は、Content-Length を過大申告しつつ本文を途中で切り接続を閉じる（毎回 readBody を
// 失敗させる真の read 失敗）テストサーバを、到達回数カウンタ付きで生成します。
func truncatingHijackServer(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
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
	return srv, &hits
}

func Test_client_Do_ResponseTooLarge(t *testing.T) {
	t.Parallel()

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("上限超過は決定的失敗なので1回しか試行しない", func(t *testing.T) {
			t.Parallel()

			srv, hits := countingLargeBodyServer(t)

			profile := httpclient.DefaultProfile()
			profile.MaxResponseBytes = 4
			profile.MaxAttempts = 3
			profile.BaseBackoff = time.Millisecond
			profile.MaxBackoff = time.Millisecond
			registry := httpclient.NewRegistry(map[httpclient.Downstream]httpclient.Profile{"tiny": profile})

			client := newClient(t, registry)
			_, err := client.Do(context.Background(), httpclient.NewRequest(httpclient.MethodGet(), "tiny", srv.URL))

			require.ErrorIs(t, err, apperror.ErrUnavailable)
			assert.Equal(t, int32(1), hits.Load()) // GET でも retry されない
		})

		t.Run("上限超過を繰り返してもbreakerはopenしない", func(t *testing.T) {
			t.Parallel()

			srv, hits := countingLargeBodyServer(t)

			profile := httpclient.DefaultProfile()
			profile.MaxResponseBytes = 4
			profile.MaxAttempts = 1
			// 1 件の失敗で open する設定。上限超過が失敗計上されないことを、以降もサーバへ到達する事実で確認する。
			profile.Breaker = httpclient.BreakerConfig{
				FailureThreshold: 0.5,
				MinRequests:      1,
				OpenDuration:     time.Hour,
				HalfOpenProbes:   1,
			}
			registry := httpclient.NewRegistry(map[httpclient.Downstream]httpclient.Profile{"tiny": profile})
			client := newClient(t, registry)

			req := httpclient.NewRequest(httpclient.MethodGet(), "tiny", srv.URL)
			const rounds = 3
			for range rounds {
				_, err := client.Do(context.Background(), req)
				require.ErrorIs(t, err, apperror.ErrUnavailable)
			}

			// breaker が open していれば fail-fast でサーバへ到達しなくなる。全リクエストが到達＝open していない。
			assert.Equal(t, int32(rounds), hits.Load())
		})

		t.Run("5xxかつ上限超過ボディでもreadBodyが先勝ちしresp_nilでretryもopenもしない", func(t *testing.T) {
			t.Parallel()

			// attempt は status 分類より先に readBody を行うため、503 とボディ上限超過が同時に起きても
			// errResponseTooLarge が先勝ちして resp=nil になる（status(503) は観測されない）。
			var hits atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				hits.Add(1)
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte("0123456789"))
			}))
			t.Cleanup(srv.Close)

			profile := httpclient.DefaultProfile()
			profile.MaxResponseBytes = 4
			profile.MaxAttempts = 3
			profile.BaseBackoff = time.Millisecond
			profile.MaxBackoff = time.Millisecond
			// 1 件の失敗で open する設定。上限超過が失敗計上されない（＝ breaker 非 open）ことも併せて固定する。
			profile.Breaker = httpclient.BreakerConfig{
				FailureThreshold: 0.5,
				MinRequests:      1,
				OpenDuration:     time.Hour,
				HalfOpenProbes:   1,
			}
			registry := httpclient.NewRegistry(map[httpclient.Downstream]httpclient.Profile{"tiny": profile})
			client := newClient(t, registry)

			req := httpclient.NewRequest(httpclient.MethodGet(), "tiny", srv.URL)
			resp, err := client.Do(context.Background(), req)
			require.ErrorIs(t, err, apperror.ErrUnavailable)
			assert.Nil(t, resp)                    // status(503) ではなく上限超過失敗が先勝ちするため resp は返らない
			assert.Equal(t, int32(1), hits.Load()) // 決定的失敗なので retry されない

			// breaker が open していれば 2 回目は fail-fast でサーバへ到達しない。到達＝ open していない。
			_, err = client.Do(context.Background(), req)
			require.ErrorIs(t, err, apperror.ErrUnavailable)
			assert.Equal(t, int32(2), hits.Load())
		})

		t.Run("真のread失敗は従来どおりリトライされる", func(t *testing.T) {
			t.Parallel()

			srv, hits := truncatingHijackServer(t)

			profile := httpclient.DefaultProfile()
			profile.MaxAttempts = 3
			profile.BaseBackoff = time.Millisecond
			profile.MaxBackoff = time.Millisecond
			profile.RetryBudgetRatio = 100
			registry := httpclient.NewRegistry(map[httpclient.Downstream]httpclient.Profile{"read": profile})

			client := newClient(t, registry)
			_, err := client.Do(context.Background(), httpclient.NewRequest(httpclient.MethodGet(), "read", srv.URL))

			require.ErrorIs(t, err, apperror.ErrUnavailable)
			assert.Equal(t, int32(3), hits.Load()) // 応答未取得の transport 失敗は retry 対象のまま
		})
	})
}

// countingLargeBodyServer は、上限超過用に固定ボディを返しつつ到達回数を数えるテストサーバを生成します。
func countingLargeBodyServer(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("0123456789"))
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

func Test_client_Do_Redirect(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("リダイレクトを追従せず3xxをErrUnavailableとして返す", func(t *testing.T) {
			t.Parallel()

			var redirectTargetHit atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/moved" {
					redirectTargetHit.Add(1)
					w.WriteHeader(http.StatusOK)
					return
				}
				w.Header().Set("Location", "/moved")
				w.WriteHeader(http.StatusFound)
			}))
			t.Cleanup(srv.Close)

			client := newClient(t, httpclient.NewRegistry(nil))
			resp, err := client.Do(context.Background(), httpclient.NewRequest(httpclient.MethodGet(), "sample", srv.URL))

			// 非追従でも resp(Location 付き)は返り、非2xx契約どおり err は ErrUnavailable。
			require.ErrorIs(t, err, apperror.ErrUnavailable)
			require.NotNil(t, resp)
			assert.Equal(t, http.StatusFound, resp.StatusCode)
			assert.Equal(t, []string{"/moved"}, resp.Header["Location"])
			assert.Equal(t, int32(0), redirectTargetHit.Load()) // 追従していない
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

func Test_client_Do_MinimumAttempt(t *testing.T) {
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
			resp, err := client.Do(context.Background(), httpclient.NewRequest(httpclient.MethodGet(), "zero", srv.URL))

			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, int32(1), hits.Load())
		})
	})
}

func Test_client_Do_Retry(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("503はMaxAttemptsまでリトライする", func(t *testing.T) {
			t.Parallel()

			srv, hits := countingServer(t, http.StatusServiceUnavailable)
			client := newClient(t, retryProfile())

			resp, err := client.Do(context.Background(), httpclient.NewRequest(httpclient.MethodGet(), "retry", srv.URL))

			require.ErrorIs(t, err, apperror.ErrUnavailable)
			require.NotNil(t, resp) // リトライ後 5xx でも resp は非nilで返る契約
			assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
			assert.Equal(t, int32(3), hits.Load())
		})

		t.Run("4xxはリトライしない", func(t *testing.T) {
			t.Parallel()

			srv, hits := countingServer(t, http.StatusBadRequest)
			client := newClient(t, retryProfile())

			_, err := client.Do(context.Background(), httpclient.NewRequest(httpclient.MethodGet(), "retry", srv.URL))

			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
			assert.Equal(t, int32(1), hits.Load())
		})

		t.Run("AllowRetryなしのPOSTは503でも1回しか飛ばない", func(t *testing.T) {
			t.Parallel()

			srv, hits := countingServer(t, http.StatusServiceUnavailable)
			client := newClient(t, retryProfile())

			_, err := client.Do(context.Background(), httpclient.NewRequest(httpclient.MethodPost(), "retry", srv.URL))

			require.ErrorIs(t, err, apperror.ErrUnavailable)
			assert.Equal(t, int32(1), hits.Load())
		})

		t.Run("AllowRetryと冪等性キーありのPOSTはリトライする", func(t *testing.T) {
			t.Parallel()

			srv, hits := countingServer(t, http.StatusServiceUnavailable)
			client := newClient(t, retryProfile())

			_, err := client.Do(context.Background(), httpclient.NewRequest(
				httpclient.MethodPost(), "retry", srv.URL,
				httpclient.WithRetry("key-1"),
			))

			require.ErrorIs(t, err, apperror.ErrUnavailable)
			assert.Equal(t, int32(3), hits.Load())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		// 「AllowRetry あり・冪等性キーなし」のランタイムガードは request_guard_internal_test.go で担保しています。
		// 公開 API から構築不能（非公開フィールドの直接設定が必要）なため、外部テストからは到達できません。

		t.Run("backoffのスリープ中にctxがキャンセルされたらErrCanceledを返し直前のResponseを破棄する", func(t *testing.T) {
			t.Parallel()

			srv, _ := countingServer(t, http.StatusServiceUnavailable)

			type callerKey struct{}
			var sleptCallerValue any

			ctrl := gomock.NewController(t)
			sleeper := mock_clock.NewMockSleeper(ctrl)
			sleeper.EXPECT().Sleep(gomock.Any(), gomock.Any()).
				DoAndReturn(func(ctx context.Context, _ time.Duration) error {
					sleptCallerValue = ctx.Value(callerKey{})
					return context.Canceled
				}).AnyTimes()

			client := httpclient.New(
				observability.NewNoopHTTPClientTransport(t),
				sleeper,
				system.NewClock(),
				retryProfile(),
				observability.NewNoopHTTPClientMetrics(t),
			)

			callerCtx := context.WithValue(context.Background(), callerKey{}, "caller")
			resp, err := client.Do(callerCtx, httpclient.NewRequest(httpclient.MethodGet(), "retry", srv.URL))

			require.ErrorIs(t, err, apperror.ErrCanceled)
			// 直前の試行で 503 の Response を得ているが、sleeper がエラーを返した経路では返さない。
			assert.Nil(t, resp)
			// 呼び出し元 ctx から派生した ctx を渡さないと、backoff 中のキャンセルを sleeper が観測できない。
			assert.Equal(t, "caller", sleptCallerValue)
		})
	})
}

func Test_client_Do_Backoff(t *testing.T) {
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
			profile.RetryBudgetRatio = 100 // budget で retry が絞られないよう十分に確保
			registry := httpclient.NewRegistry(map[httpclient.Downstream]httpclient.Profile{"retry": profile})

			var slept []time.Duration
			ctrl := gomock.NewController(t)
			sleeper := mock_clock.NewMockSleeper(ctrl)
			sleeper.EXPECT().Sleep(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, d time.Duration) error {
					slept = append(slept, d)
					return nil
				}).AnyTimes()

			client := httpclient.New(
				observability.NewNoopHTTPClientTransport(t),
				sleeper,
				system.NewClock(),
				registry,
				observability.NewNoopHTTPClientMetrics(t),
			)
			_, err := client.Do(context.Background(), httpclient.NewRequest(httpclient.MethodGet(), "retry", srv.URL))

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

func Test_client_Do_Deadline(t *testing.T) {
	t.Parallel()

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("overallデッドラインを超えるリトライは打ち切られMaxAttemptsに達しない", func(t *testing.T) {
			t.Parallel()

			srv, hits := countingServer(t, http.StatusServiceUnavailable)

			profile := httpclient.DefaultProfile()
			profile.MaxAttempts = 20
			// backoff は極小にして jitter の振れ幅を打ち切り判定に対して無視できるようにし、
			// 打ち切りは fake clock の固定 step（20s/サイクル）と overall deadline(60s) の関係のみで決まる。
			profile.BaseBackoff = time.Millisecond
			profile.MaxBackoff = time.Millisecond
			profile.OverallTimeout = 60 * time.Second
			profile.PerAttemptTimeout = 10 * time.Second // 既定 3s より広く取り、実時間側を打ち切り要因から外す
			// retry budget が deadline より先に打ち切らないよう十分なトークンを与え、
			// 打ち切り要因を overall deadline に限定する。
			profile.RetryBudgetRatio = 10
			profile.Breaker.MinRequests = 1 << 30 // breaker は開かせない（DefaultProfile の閾値に依存させない）
			registry := httpclient.NewRegistry(map[httpclient.Downstream]httpclient.Profile{"retry": profile})

			// Sleep で fake 時刻を 20s/サイクル進める clock を注入し、打ち切りを注入 clock 基準の
			// canRetryWithin（overallDeadline 到達）のみに担わせる（jitter 非依存）。
			// overall(60s) / per-attempt(10s) の実時間 ctx タイマーは、fake sleep が実待機せず往復先も
			// httptest サーバのためテストが ms オーダーで完了し、発火しない。
			// 開始時刻は実時刻と混同しないよう遠未来に置く。
			fakeClock := clocktestkit.NewStepClock(time.Now().Add(time.Hour), 20*time.Second)
			client := httpclient.New(
				observability.NewNoopHTTPClientTransport(t),
				fakeClock,
				fakeClock,
				registry,
				observability.NewNoopHTTPClientMetrics(t),
			)
			resp, err := client.Do(context.Background(), httpclient.NewRequest(httpclient.MethodGet(), "retry", srv.URL))

			require.ErrorIs(t, err, apperror.ErrUnavailable)
			// 非nil が除外できるのは sleeper error 経路（resp=nil で return）だけで、breaker open は
			// 直前の resp を返すため弁別にならない（breaker 非関与は上の MinRequests が保証する）。
			// 打ち切りが canRetryWithin（deadline 到達）由来であることの弁別は下の hits 厳密一致が担う。
			require.NotNil(t, resp)
			assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
			// 20s/step × 3 backoff で fakeNow がちょうど deadline(60s)に達し、hits は決定的に 4（jitter 非依存）。
			assert.Equal(t, int32(4), hits.Load())
		})
	})
}

func Test_client_Do_Breaker(t *testing.T) {
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

			req := httpclient.NewRequest(httpclient.MethodGet(), "brk", srv.URL)

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

func Test_client_Do_BreakerReturnsLastResponse(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("retry2回目でbreakerがopenでも直前のレスポンスを返す", func(t *testing.T) {
			t.Parallel()

			srv, hits := countingServer(t, http.StatusServiceUnavailable)

			profile := httpclient.DefaultProfile()
			profile.MaxAttempts = 2 // attempt1 で open させ attempt2 の allow=false を踏む
			profile.BaseBackoff = time.Millisecond
			profile.MaxBackoff = time.Millisecond
			profile.RetryBudgetRatio = 100 // budget で retry が絞られないよう確保
			profile.Breaker = httpclient.BreakerConfig{
				FailureThreshold: 0.5,
				MinRequests:      1,         // 1 件の失敗で即 open
				OpenDuration:     time.Hour, // attempt2 までに half-open へ遷移させない
				HalfOpenProbes:   1,
			}
			registry := httpclient.NewRegistry(map[httpclient.Downstream]httpclient.Profile{"brk": profile})
			client := newClient(t, registry)

			resp, err := client.Do(context.Background(), httpclient.NewRequest(httpclient.MethodGet(), "brk", srv.URL))

			// attempt1 の 503 で open。attempt2 は allow=false だが直前 resp を返す。
			require.ErrorIs(t, err, apperror.ErrUnavailable)
			require.NotNil(t, resp)
			assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
			assert.Equal(t, int32(1), hits.Load()) // attempt2 はサーバへ到達しない
		})
	})
}

func Test_client_Do_Budget(t *testing.T) {
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

			req := httpclient.NewRequest(httpclient.MethodGet(), "bdg", srv.URL)

			// 初期トークン(=retryBudgetInitialTokens)を消費し切るまでリクエストを繰り返す。
			for range 2 {
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

func Test_client_Do_RetryAfter(t *testing.T) {
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

			client := httpclient.New(
				observability.NewNoopHTTPClientTransport(t),
				sleeper,
				system.NewClock(),
				registry,
				observability.NewNoopHTTPClientMetrics(t),
			)
			_, err := client.Do(context.Background(), httpclient.NewRequest(httpclient.MethodGet(), "ra", srv.URL))

			require.ErrorIs(t, err, apperror.ErrUnavailable)
			require.Len(t, slept, 1)
			assert.Equal(t, 2*time.Second, slept[0]) // バックオフ(<=20ms)ではなく Retry-After(2s)
		})
	})
}

func Test_client_Do_Timeout(t *testing.T) {
	t.Parallel()

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("per-attemptタイムアウト超過はErrUnavailableを返す", func(t *testing.T) {
			t.Parallel()

			// per-attempt タイムアウトでリクエスト context がキャンセルされるまでブロックするサーバ。
			var hits atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				hits.Add(1)
				<-r.Context().Done()
			}))
			t.Cleanup(srv.Close)

			profile := httpclient.DefaultProfile()
			// per-attempt は接続確立に十分な余裕を持たせる（短すぎるとサーバ到達前に発火し hits=0 で
			// フレークする）。overall とは大きく離し、per-attempt 単独の打ち切りを経過時間で分離検証する
			// （overall で打ち切られる構成だと per-attempt を外しても同じ ErrUnavailable が返り区別できない）。
			profile.PerAttemptTimeout = 200 * time.Millisecond
			profile.OverallTimeout = 5 * time.Second
			profile.MaxAttempts = 1
			registry := httpclient.NewRegistry(map[httpclient.Downstream]httpclient.Profile{"slow": profile})

			client := newClient(t, registry)
			start := time.Now()
			resp, err := client.Do(context.Background(), httpclient.NewRequest(httpclient.MethodGet(), "slow", srv.URL))
			elapsed := time.Since(start)

			require.ErrorIs(t, err, apperror.ErrUnavailable)
			assert.Nil(t, resp)
			assert.Equal(t, int32(1), hits.Load())
			// per-attempt(200ms) が打ち切り要因であることを、overall(5s) より大幅に短い経過時間で確認する
			// （上限は per-attempt の 10 倍・overall の半分未満に取り、負荷変動に耐える）。
			assert.Less(t, elapsed, 2*time.Second)
		})

		t.Run("overallタイムアウト超過はper-attempt発火前にErrUnavailableを返す", func(t *testing.T) {
			t.Parallel()

			// overall タイムアウトでリクエスト context がキャンセルされるまでブロックするサーバ。
			var hits atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				hits.Add(1)
				<-r.Context().Done()
			}))
			t.Cleanup(srv.Close)

			profile := httpclient.DefaultProfile()
			// per-attempt ケースの対称形。overall は接続確立に十分な余裕を持たせ（短すぎるとサーバ到達前に
			// 発火し hits=0 でフレークする）、per-attempt とは大きく離して overall 単独の打ち切りを
			// 経過時間で分離検証する（MaxAttempts=1 のため retry は発生しない）。
			profile.OverallTimeout = 200 * time.Millisecond
			profile.PerAttemptTimeout = 5 * time.Second
			profile.MaxAttempts = 1
			registry := httpclient.NewRegistry(map[httpclient.Downstream]httpclient.Profile{"slow": profile})

			client := newClient(t, registry)
			start := time.Now()
			resp, err := client.Do(context.Background(), httpclient.NewRequest(httpclient.MethodGet(), "slow", srv.URL))
			elapsed := time.Since(start)

			require.ErrorIs(t, err, apperror.ErrUnavailable)
			assert.Nil(t, resp)
			assert.Equal(t, int32(1), hits.Load())
			// overall(200ms) が打ち切り要因であることを、per-attempt(5s) より大幅に短い経過時間で確認する
			// （上限は overall の 10 倍・per-attempt の半分未満に取り、負荷変動に耐える）。
			assert.Less(t, elapsed, 2*time.Second)
		})
	})
}
