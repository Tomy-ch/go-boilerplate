// Package hook は、サーバーのライフサイクルフックを提供します。
package hook

import (
	"context"
	"sync"
	"time"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/httpstack/ratelimit"
	"go-boilerplate/internal/di/lifecycle"
	"go-boilerplate/internal/logging"
)

const ratelimitCallerSkip = 3

type rateLimitCleanupHook struct {
	rl     ratelimit.IPRateLimiter
	logger logging.Logger
	osCfg  *config.OperationSystemConfig
	ipCfg  *config.IPRateLimitConfig

	mu     sync.Mutex
	stopCh chan struct{}
	once   sync.Once
}

func RegisterRateLimitHooks(
	reg lifecycle.Registrar,
	rl ratelimit.IPRateLimiter,
	logger logging.Logger,
	osCfg *config.OperationSystemConfig,
	ipCfg *config.IPRateLimitConfig,
) {
	if !ipCfg.Enabled() {
		logger.Named("ratelimit.Hooks").CallerSkip(ratelimitCallerSkip).Info(
			"RateLimiter disabled",
			logging.String(logging.EventTypeKey, logging.EventTypeStart),
			logging.Time(logging.EventAtKey, time.Now()),
			logging.String(logging.EventTzKey, osCfg.TimeZone()),
		)
		return
	}

	newRateLimitCleanupHook(rl, logger, osCfg, ipCfg).Register(reg)
}

func newRateLimitCleanupHook(
	rl ratelimit.IPRateLimiter,
	logger logging.Logger,
	osCfg *config.OperationSystemConfig,
	ipCfg *config.IPRateLimitConfig,
) *rateLimitCleanupHook {
	return &rateLimitCleanupHook{
		rl:     rl,
		logger: logger,
		osCfg:  osCfg,
		ipCfg:  ipCfg,
	}
}

// Register は lifecycle.Registrar に start/stop を登録します。
func (h *rateLimitCleanupHook) Register(reg lifecycle.Registrar) {
	reg.RegisterStart(h.onStart)
	reg.RegisterStop(h.onStop)
}

// onStart は、レートリミットのクリーンアップゴルーチンを開始します。
func (h *rateLimitCleanupHook) onStart(startCtx context.Context) error {
	h.mu.Lock()
	h.stopCh = make(chan struct{})
	h.once = sync.Once{}
	stopCh := h.stopCh
	h.mu.Unlock()

	ticker := time.NewTicker(h.ipCfg.CleanupInterval())

	h.logger.Named("ratelimit.Hooks").CallerSkip(ratelimitCallerSkip).Info(
		"RateLimiter cleanup started",
		logging.String(logging.EventTypeKey, logging.EventTypeStart),
		logging.Time(logging.EventAtKey, time.Now()),
		logging.String(logging.EventTzKey, h.osCfg.TimeZone()),
		logging.Bool("enabled", h.ipCfg.Enabled()),
		logging.DurationMs("cleanup_interval", h.ipCfg.CleanupInterval()),
		logging.DurationMs("ttl", h.ipCfg.TTL()),
		logging.Int("burst", h.ipCfg.Burst()),
	)

	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				h.rl.Cleanup()
			case <-startCtx.Done():
				h.logger.CallerSkip(ratelimitCallerSkip).Named("ratelimit.Hooks").Info(
					"RateLimiter cleanup stopped (context done)",
					logging.String(logging.EventTypeKey, logging.EventTypeEnd),
					logging.Time(logging.EventAtKey, time.Now()),
					logging.String(logging.EventTzKey, h.osCfg.TimeZone()),
				)
				return

			case <-stopCh:
				h.logger.CallerSkip(ratelimitCallerSkip).Named("ratelimit.Hooks").Info(
					"RateLimiter cleanup stopped",
					logging.String(logging.EventTypeKey, logging.EventTypeEnd),
					logging.Time(logging.EventAtKey, time.Now()),
					logging.String(logging.EventTzKey, h.osCfg.TimeZone()),
				)
				return
			}
		}
	}()

	return nil
}

// onStop は、レートリミットのクリーンアップゴルーチンを停止します。
func (h *rateLimitCleanupHook) onStop(_ context.Context) error {
	h.mu.Lock()
	stopCh := h.stopCh
	h.mu.Unlock()

	if stopCh != nil {
		h.once.Do(func() {
			close(stopCh)
		})
	}
	return nil
}
