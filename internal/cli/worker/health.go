package worker

import (
	"context"
	"errors"
	"net/http"
	"time"

	"go-boilerplate/internal/logging"
)

const (
	healthReadHeaderTimeout = 5 * time.Second
	healthReadTimeout       = 10 * time.Second
	healthWriteTimeout      = 10 * time.Second
	healthIdleTimeout       = 60 * time.Second
)

// NewHealthServer は、liveness(/healthz) と readiness(/readyz) を公開する health listener の
// 開始・終了関数を返します。ready は readiness 判定（例: engine.Healthy）を渡します。
// metrics サーバーと異なり専用の ServeMux を使い、pprof/metrics と衝突しません。
func NewHealthServer(addr string, ready func() bool, logger logging.Logger) (func(), func(context.Context)) {
	srv := &http.Server{
		Addr:              addr,
		Handler:           healthMux(ready),
		ReadHeaderTimeout: healthReadHeaderTimeout,
		ReadTimeout:       healthReadTimeout,
		WriteTimeout:      healthWriteTimeout,
		IdleTimeout:       healthIdleTimeout,
	}

	start := func() {
		go func() {
			if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				logger.Named("worker.health").Error("health server error", logging.Error(logging.ErrorKey, err))
			}
		}()
	}
	stop := func(ctx context.Context) {
		if err := srv.Shutdown(ctx); err != nil {
			logger.Named("worker.health").Error("health server shutdown error", logging.Error(logging.ErrorKey, err))
		}
	}
	return start, stop
}

// healthMux は、liveness(/healthz) と readiness(/readyz) を提供する ServeMux を構築します。
func healthMux(ready func() bool) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		if ready() {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ready"))
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("not ready"))
	})
	return mux
}
