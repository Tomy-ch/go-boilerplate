package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"boilerplate-go/internal/config"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func TestNewIPRateLimiter(t *testing.T) {
	t.Parallel()

	delta := float64(0.0001)

	cfg := config.MockConfigForTest(t)
	ipCfg := config.NewIPRateLimitConfig(cfg)
	rl := NewIPRateLimiter(ipCfg)
	require.NotNil(t, rl)

	irl, ok := rl.(*iPRateLimiter)
	require.True(t, ok)
	require.Equal(t, ipCfg, irl.cfg)
	require.InEpsilon(t, float64(irl.limit), ipCfg.Limit(), delta)
	require.Empty(t, irl.entries)
}

func Test_iPRateLimiter_resolveClientIP(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	ipCfg := config.NewIPRateLimitConfig(cfg)
	rl := &iPRateLimiter{
		cfg:     ipCfg,
		limit:   rate.Limit(ipCfg.Limit()),
		entries: make(map[string]*limiterEntry),
	}

	t.Run("RealIPが空の場合、unknownを返す", func(t *testing.T) {
		t.Parallel()
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = ""
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// no IP -> unknown
		ip := rl.resolveClientIP(c)
		require.Equal(t, "unknown", ip)
	})

	t.Run("RealIPが設定されている場合、その値を返す", func(t *testing.T) {
		t.Parallel()

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(echo.HeaderXRealIP, "1.2.3.4")
		c := e.NewContext(req, httptest.NewRecorder())
		ip := rl.resolveClientIP(c)
		require.Equal(t, "1.2.3.4", ip)
	})
}

func Test_iPRateLimiter_ensureLimiter(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	ipCfg := config.NewIPRateLimitConfig(cfg)
	rl := &iPRateLimiter{
		cfg:     ipCfg,
		limit:   rate.Limit(ipCfg.Limit()),
		entries: make(map[string]*limiterEntry),
	}

	lim := rl.ensureLimiter("ip1", rate.Limit(5), 2)

	t.Run("新しいIPアドレスの場合、新しいlimiterが作成される", func(t *testing.T) {
		t.Parallel()
		require.NotNil(t, lim)
		require.Contains(t, rl.entries, "ip1")
	})

	t.Run("既存のIPアドレスの場合、既存のlimiterが返される", func(t *testing.T) {
		t.Parallel()
		old := rl.entries["ip1"].lastSeen
		time.Sleep(5 * time.Millisecond)
		lim2 := rl.ensureLimiter("ip1", rate.Limit(5), 2)
		require.Equal(t, lim, lim2)
		require.True(t, rl.entries["ip1"].lastSeen.After(old))
	})
}

func Test_iPRateLimiter_AllowRequest(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	ipCfg := config.NewIPRateLimitConfig(cfg)
	rl := &iPRateLimiter{
		cfg:     ipCfg,
		limit:   rate.Limit(ipCfg.Limit()),
		entries: make(map[string]*limiterEntry),
	}

	e := echo.New()

	t.Run("IPRateLimiterのEnabledがtrueの場合、limiterのAllow結果が返される", func(t *testing.T) {
		t.Parallel()
		rl.limit = rate.Limit(1000)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(echo.HeaderXRealIP, "10.0.0.1")
		c := e.NewContext(req, httptest.NewRecorder())
		ok := rl.AllowRequest(c)
		require.True(t, ok)
	})

	t.Run("IPRateLimiterのEnabledがfalseの場合、常にtrueが返される", func(t *testing.T) {
		t.Parallel()
		rl.entries["10.0.0.2"] = &limiterEntry{limiter: rate.NewLimiter(0, 0), lastSeen: time.Now()}
		req2 := httptest.NewRequest(http.MethodGet, "/", nil)
		req2.Header.Set(echo.HeaderXRealIP, "10.0.0.2")
		c2 := e.NewContext(req2, httptest.NewRecorder())
		ok2 := rl.AllowRequest(c2)
		require.False(t, ok2)
	})
}

func Test_iPRateLimiter_Cleanup(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	ipCfg := config.NewIPRateLimitConfig(cfg)

	t.Run("IPRateLimiterのEnabledがfalseの場合、何もしない", func(t *testing.T) {
		t.Parallel()

		rl := iPRateLimiter{
			cfg:     ipCfg,
			limit:   rate.Limit(ipCfg.Limit()),
			entries: make(map[string]*limiterEntry),
		}

		rl.cfg = config.NewIPRateLimitConfig(&config.Config{}) // Disabled
		rl.entries["someip"] = &limiterEntry{
			limiter:  rate.NewLimiter(1, 1),
			lastSeen: time.Now().Add(-10 * time.Minute),
		}

		rl.Cleanup()
		_, ok := rl.entries["someip"]
		require.True(t, ok)
	})

	t.Run("IPRateLimiterのEnabledがtrueの場合、TTLを過ぎたエントリが削除される", func(t *testing.T) {
		t.Parallel()

		rl := iPRateLimiter{
			cfg:     ipCfg,
			limit:   rate.Limit(ipCfg.Limit()),
			entries: make(map[string]*limiterEntry),
		}

		rl.entries["old"] = &limiterEntry{
			limiter:  rate.NewLimiter(1, 1),
			lastSeen: time.Now().Add(-10 * rl.cfg.TTL()),
		}
		rl.entries["new"] = &limiterEntry{
			limiter:  rate.NewLimiter(1, 1),
			lastSeen: time.Now(),
		}

		rl.Cleanup()
		_, okOld := rl.entries["old"]
		_, okNew := rl.entries["new"]
		require.False(t, okOld)
		require.True(t, okNew)
	})
}
