//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE -package=mock_$GOPACKAGE

package ratelimit

import (
	"sync"
	"time"

	"boilerplate-go/internal/config"

	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
)

type IPRateLimiter interface {
	AllowRequest(c echo.Context) bool
	Cleanup()
}

// iPRateLimiter は、IPアドレスごとのレートリミッターを提供します。
//
// この実装はインメモリであり、単一プロセス内での使用を想定しています。
// 大量のIPアドレスを扱う場合や、分散環境での使用には適していません。
// 複数のレプリカへの並行スケール時は、この機能を外してRedis等の外部ストアを利用することを検討してください。
type iPRateLimiter struct {
	cfg     *config.IPRateLimitConfig
	limit   rate.Limit
	entries map[string]*limiterEntry

	mu sync.Mutex
}

// limiterEntry は IP ごとの limiter を保持します。
type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewIPRateLimiter は、IPアドレスごとのレートリミッターを生成します。
func NewIPRateLimiter(ipLimitCfg *config.IPRateLimitConfig) IPRateLimiter {
	return &iPRateLimiter{
		cfg:     ipLimitCfg,
		limit:   rate.Limit(ipLimitCfg.Limit()),
		entries: make(map[string]*limiterEntry),
	}
}

// AllowRequest は、このリクエストを通してよいか（= rate limit を満たすか）を判定します。
// 返り値が false のときは、呼び出し側で 429 応答を返してください。
func (rl *iPRateLimiter) AllowRequest(c echo.Context) bool {
	ip := rl.resolveClientIP(c)
	lim := rl.ensureLimiter(ip, rl.limit, rl.cfg.Burst())
	return lim.Allow()
}

// Cleanup は TTL を過ぎた IP エントリを削除します。
func (rl *iPRateLimiter) Cleanup() {
	if !rl.cfg.Enabled() {
		return
	}

	now := time.Now()
	rl.mu.Lock()
	defer rl.mu.Unlock()

	for ip, e := range rl.entries {
		if now.Sub(e.lastSeen) > rl.cfg.TTL() {
			delete(rl.entries, ip)
		}
	}
}

// resolveClientIP は、クライアントのIPアドレスを取得します。
func (rl *iPRateLimiter) resolveClientIP(c echo.Context) string {
	ip := c.RealIP()
	if ip == "" {
		return "unknown"
	}
	return ip
}

// ensureLimiter は、IPアドレスに対応する rate.Limiter を取得または作成します。
func (rl *iPRateLimiter) ensureLimiter(ip string, limit rate.Limit, burst int) *rate.Limiter {
	now := time.Now()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	if e, ok := rl.entries[ip]; ok {
		e.lastSeen = now
		return e.limiter
	}

	l := rate.NewLimiter(limit, burst)
	rl.entries[ip] = &limiterEntry{
		limiter:  l,
		lastSeen: now,
	}
	return l
}
