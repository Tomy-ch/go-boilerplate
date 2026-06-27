package worker

import (
	"time"

	"go-boilerplate/pkg/backoff"
)

const (
	// defaultDrainTimeout は、DrainTimeout 未設定時の既定値です。
	defaultDrainTimeout = 30 * time.Second
	// defaultProgressStaleAfter は、ProgressStaleAfter 未設定時の既定値です。
	defaultProgressStaleAfter = 60 * time.Second
	// circuitBackoffMultiplier は、Open の cooldown を指数的に伸ばす際の倍率です。
	circuitBackoffMultiplier = 2

	// defaultNackBackoffInitial は、NackBackoffInitial 未設定時の既定値です。
	defaultNackBackoffInitial = 1 * time.Second
	// defaultNackBackoffMax は、NackBackoffMax 未設定時の既定値です。
	defaultNackBackoffMax = 30 * time.Second
	// nackBackoffMultiplier は、per-message 再配送 backoff を指数的に伸ばす際の倍率です。
	nackBackoffMultiplier = 2
)

// Settings は、engine の挙動を制御する engine-core 設定です（broker 非依存）。
type Settings struct {
	// Concurrency は、同時に Handle を実行する最大数です（B1）。
	Concurrency int
	// MaxInFlight は、受信済み・未確定（Ack/Nack 前）の最大メッセージ数です（B2）。
	MaxInFlight int
	// BatchSize は、1 回の Receive で取得する最大件数です。
	BatchSize int
	// ExtendInterval は、長時間処理中に Extend を呼ぶ周期です（0 以下で無効）（A3）。
	ExtendInterval time.Duration
	// DrainTimeout は、停止時に in-flight の完了を待つ上限です。
	DrainTimeout time.Duration
	// ReceiveCountWarnThreshold は、この回数以上の再配送で warn する閾値です（0 以下で無効）（A7）。
	ReceiveCountWarnThreshold int

	// CircuitFailureThreshold は、Open に入るまでの連続失敗数です（0 以下でサーキット無効）。
	CircuitFailureThreshold int
	// CircuitOpenBackoffInitial は、Open の初回 cooldown です。
	CircuitOpenBackoffInitial time.Duration
	// CircuitOpenBackoffMax は、Open の cooldown 上限です。
	CircuitOpenBackoffMax time.Duration
	// CircuitHalfOpenProbe は、Half-open 時に試行する最大件数です。
	CircuitHalfOpenProbe int
	// ProgressStaleAfter は、readiness 判定で「進捗なし(stuck)」とみなすまでの時間です。
	ProgressStaleAfter time.Duration

	// NackBackoffInitial は、retryable 失敗時の per-message 再配送 backoff の初回待機です。
	NackBackoffInitial time.Duration
	// NackBackoffMax は、per-message 再配送 backoff の上限です。
	NackBackoffMax time.Duration
}

// normalize は、ゼロ値に安全な既定値を補います。
func (s *Settings) normalize() {
	if s.Concurrency < 1 {
		s.Concurrency = 1
	}
	if s.MaxInFlight < s.Concurrency {
		s.MaxInFlight = s.Concurrency
	}
	if s.BatchSize < 1 {
		s.BatchSize = 1
	}
	if s.BatchSize > s.MaxInFlight {
		s.BatchSize = s.MaxInFlight
	}
	if s.DrainTimeout <= 0 {
		s.DrainTimeout = defaultDrainTimeout
	}
	if s.CircuitHalfOpenProbe < 1 {
		s.CircuitHalfOpenProbe = 1
	}
	if s.ProgressStaleAfter <= 0 {
		s.ProgressStaleAfter = defaultProgressStaleAfter
	}
	if s.NackBackoffInitial <= 0 {
		s.NackBackoffInitial = defaultNackBackoffInitial
	}
	if s.NackBackoffMax <= 0 {
		s.NackBackoffMax = defaultNackBackoffMax
	}
}

// circuitBackoff は、Settings から Open の cooldown 算出器を構築します。
func (s *Settings) circuitBackoff() backoff.Exponential {
	return backoff.Exponential{
		Initial:    s.CircuitOpenBackoffInitial,
		Max:        s.CircuitOpenBackoffMax,
		Multiplier: circuitBackoffMultiplier,
	}
}

// nackBackoff は、Settings から per-message 再配送 backoff の算出器を構築します。
func (s *Settings) nackBackoff() backoff.Exponential {
	return backoff.Exponential{
		Initial:    s.NackBackoffInitial,
		Max:        s.NackBackoffMax,
		Multiplier: nackBackoffMultiplier,
	}
}
