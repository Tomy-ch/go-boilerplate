package worker

import (
	"sync"
	"time"

	"go-boilerplate/pkg/backoff"
)

const (
	// phaseClosed は通常状態（通常 poll）。
	phaseClosed circuitPhase = iota
	// phaseOpen は intake 停止（cooldown 中は Receive を呼ばない）。
	phaseOpen
	// phaseHalfOpen は限定試行（probe 件だけ Receive）。
	phaseHalfOpen
)

// circuitPhase は、サーキットブレーカの状態です。
type circuitPhase int

// circuit は、下流失敗継続時に intake を止める 3 状態サーキットブレーカです。
// 信号は handler の Retryable 連続失敗と poll エラー（onFailure）。成功と Permanent は onSuccess。
type circuit struct {
	mu sync.Mutex

	phase       circuitPhase
	probing     bool // Half-open で probe バッチを投入済み（結果待ち）なら true
	failures    int
	threshold   int // 0 以下でサーキット無効
	openCount   int
	curCooldown time.Duration
	backoff     backoff.Exponential
}

// newCircuit は、サーキットブレーカを生成します。threshold が 0 以下なら常に Closed のままです。
func newCircuit(threshold int, bo backoff.Exponential) *circuit {
	return &circuit{threshold: threshold, backoff: bo}
}

// phaseNow は、現在の状態を返します。
func (c *circuit) phaseNow() circuitPhase {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.phase
}

// cooldown は、現在の Open エピソードの cooldown を返します。
func (c *circuit) cooldown() time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.curCooldown
}

// toHalfOpen は、Open から Half-open へ遷移します（cooldown 経過後に poll loop が呼ぶ）。
// 新しい Half-open エピソードなので probing は未投入（false）へ戻します。
func (c *circuit) toHalfOpen() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.phase == phaseOpen {
		c.phase = phaseHalfOpen
		c.probing = false
	}
}

// abortProbe は、Half-open の probe バッチが空振り（0 件受信）だったとき probing を解除します。
// probe が in-flight にならないと結果（onSuccess/onFailure）が出ず slotFreed も鳴らないため、
// これを解除しないと poll loop が次周の acquire で永久停止する。解除後は次周で再 probe できる。
func (c *circuit) abortProbe() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.phase == phaseHalfOpen {
		c.probing = false
	}
}

// tryBeginProbe は、Half-open で probe バッチをまだ投入していなければ probing を確定して true を返します。
// 既に probing 中（結果待ち）または Half-open でない場合は false を返し、呼び出し側は新規 Receive を止めます。
// これにより 1 Half-open エピソードで投入する probe は 1 バッチ（総試行数 ≤ CircuitHalfOpenProbe）に限定されます。
func (c *circuit) tryBeginProbe() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.phase == phaseHalfOpen && !c.probing {
		c.probing = true
		return true
	}
	return false
}

// onSuccess は、成功（または Permanent 退避完了）を記録します。Half-open なら Closed へ復帰します。
func (c *circuit) onSuccess() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failures = 0
	if c.phase == phaseHalfOpen {
		c.phase = phaseClosed
		c.probing = false
		c.openCount = 0
	}
}

// onFailure は、Retryable 失敗または poll エラーを記録します。閾値超過 or Half-open 失敗で Open へ遷移します。
func (c *circuit) onFailure() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.threshold <= 0 {
		return
	}
	c.failures++
	switch c.phase {
	case phaseHalfOpen:
		c.trip()
	case phaseClosed:
		if c.failures >= c.threshold {
			c.trip()
		}
	case phaseOpen:
		// Open 中は Receive しないため通常は到達しない。
	}
}

// trip は、Open へ遷移し cooldown を確定します（mu ロック下で呼ぶこと）。
func (c *circuit) trip() {
	c.phase = phaseOpen
	c.probing = false
	c.curCooldown = c.backoff.Duration(c.openCount)
	c.openCount++
}
