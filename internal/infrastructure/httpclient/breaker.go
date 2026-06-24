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

// breakerState は、circuit breaker の状態を表します。
type breakerState int

// breaker は、Downstream ごとの circuit breaker です。
type breaker struct {
	mu     sync.Mutex
	config BreakerConfig
	state  breakerState

	// generation は状態遷移ごとに増加し、allow で発行したプローブ枠と record の整合を取るためのエポック識別子です。
	generation uint64

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

// allow は、now 時点でリクエストを通してよいかと、その試行が属するエポックを返します。
// 返したエポックは record にそのまま渡します。
// open は OpenDuration 経過で half-open へ遷移し、half-open は HalfOpenProbes 件までプローブを通します。
func (b *breaker) allow(now time.Time) (bool, uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case breakerOpen:
		if now.Sub(b.openedAt) >= b.config.OpenDuration {
			b.toHalfOpen()
			b.halfOpenProbes++
			return true, b.generation
		}
		return false, b.generation
	case breakerHalfOpen:
		if b.halfOpenProbes < b.config.HalfOpenProbes {
			b.halfOpenProbes++
			return true, b.generation
		}
		return false, b.generation
	default: // breakerClosed
		return true, b.generation
	}
}

// record は、試行結果を記録し状態遷移を行います。success=false は downstream 起因の失敗です。
func (b *breaker) record(success bool, now time.Time, generation uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case breakerHalfOpen:
		// half-open は厳密にプローブ数で回復判定するため、別エポックの遅延結果を無視する。
		if generation != b.generation {
			return
		}
		if !success {
			b.toOpen(now)
			return
		}
		b.halfOpenSuccess++
		if b.halfOpenSuccess >= b.config.HalfOpenProbes {
			b.toClosed()
		}
	case breakerClosed:
		// closed の集計は状態遷移時にリセットされるため、競合した遅延 record も無害に計上できる。
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
	b.generation++
	b.halfOpenProbes = 0
	b.halfOpenSuccess = 0
}

func (b *breaker) toHalfOpen() {
	b.state = breakerHalfOpen
	b.generation++
	b.halfOpenProbes = 0
	b.halfOpenSuccess = 0
}

func (b *breaker) toClosed() {
	b.state = breakerClosed
	b.generation++
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
