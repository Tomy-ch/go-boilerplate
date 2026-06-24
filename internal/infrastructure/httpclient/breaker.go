package httpclient

import (
	"sync"
	"time"
)

const (
	// breakerClosed は、通常状態（リクエストを通す）です。
	breakerClosed breakerState = iota
	// breakerHalfOpen は、回復確認のためプローブを限定的に通す状態です。
	breakerHalfOpen
	// breakerOpen は、即 fail-fast する状態です。
	breakerOpen
)

// breakerState は、circuit breaker の状態です（0:closed 1:half-open 2:open）。
type breakerState int

// breaker は、per-downstream の circuit breaker です（A-6）。
// 時刻は now 引数で注入し、テストを決定的にします（実呼び出し側は time.Now() を渡します）。
type breaker struct {
	mu     sync.Mutex
	config BreakerConfig
	state  breakerState

	requests int
	failures int

	openedAt        time.Time
	halfOpenProbes  int
	halfOpenSuccess int
}

// breakerManager は、Downstream ごとの breaker を保持します。
type breakerManager struct {
	mu       sync.Mutex
	breakers map[Downstream]*breaker
}

// newBreaker は、closed 状態の breaker を生成します。
func newBreaker(config BreakerConfig) *breaker {
	return &breaker{config: config, state: breakerClosed}
}

// allow は、now 時点でリクエストを通してよいかを返します。
// open は OpenDuration 経過で half-open へ遷移し、half-open は HalfOpenProbes 件までプローブを通します。
func (b *breaker) allow(now time.Time) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case breakerOpen:
		if now.Sub(b.openedAt) >= b.config.OpenDuration {
			b.toHalfOpen()
			b.halfOpenProbes++
			return true
		}
		return false
	case breakerHalfOpen:
		if b.halfOpenProbes < b.config.HalfOpenProbes {
			b.halfOpenProbes++
			return true
		}
		return false
	default: // breakerClosed
		return true
	}
}

// record は、試行結果を記録し状態遷移を行います。success=false は downstream 起因の失敗です。
func (b *breaker) record(success bool, now time.Time) {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case breakerHalfOpen:
		if !success {
			b.toOpen(now)
			return
		}
		b.halfOpenSuccess++
		if b.halfOpenSuccess >= b.config.HalfOpenProbes {
			b.toClosed()
		}
	case breakerClosed:
		b.requests++
		if !success {
			b.failures++
		}
		if b.shouldOpen() {
			b.toOpen(now)
		}
	default: // breakerOpen（allow が遷移を司るため記録は無視）
	}
}

// currentState は、現在の状態を返します。
func (b *breaker) currentState() breakerState {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.state
}

func (b *breaker) shouldOpen() bool {
	if b.requests < b.config.MinRequests {
		return false
	}
	rate := float64(b.failures) / float64(b.requests)
	return rate >= b.config.FailureThreshold
}

func (b *breaker) toOpen(now time.Time) {
	b.state = breakerOpen
	b.openedAt = now
	b.halfOpenProbes = 0
	b.halfOpenSuccess = 0
}

func (b *breaker) toHalfOpen() {
	b.state = breakerHalfOpen
	b.halfOpenProbes = 0
	b.halfOpenSuccess = 0
}

func (b *breaker) toClosed() {
	b.state = breakerClosed
	b.requests = 0
	b.failures = 0
	b.halfOpenProbes = 0
	b.halfOpenSuccess = 0
}

// newBreakerManager は、breakerManager を生成します。
func newBreakerManager() *breakerManager {
	return &breakerManager{breakers: make(map[Downstream]*breaker)}
}

// get は、d に対応する breaker を返します（無ければ config で新規生成）。
func (m *breakerManager) get(d Downstream, config BreakerConfig) *breaker {
	m.mu.Lock()
	defer m.mu.Unlock()

	b, ok := m.breakers[d]
	if !ok {
		b = newBreaker(config)
		m.breakers[d] = b
	}
	return b
}
