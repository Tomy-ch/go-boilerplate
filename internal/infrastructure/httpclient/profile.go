package httpclient

import "time"

const (
	// デフォルト Profile（未登録 Downstream 用の安全側設定）の各値。
	defaultPerAttemptTimeout = 3 * time.Second
	defaultOverallTimeout    = 10 * time.Second
	defaultMaxAttempts       = 3
	defaultBaseBackoff       = 100 * time.Millisecond
	defaultMaxBackoff        = 2 * time.Second
	defaultRetryBudgetRatio  = 0.1
	defaultMaxResponseBytes  = 4 << 20 // 4 MiB
	defaultPropagateTrace    = true

	// デフォルト BreakerConfig の各値。
	defaultBreakerFailureThreshold = 0.5
	defaultBreakerMinRequests      = 20
	defaultBreakerOpenDuration     = 5 * time.Second
	defaultBreakerHalfOpenProbes   = 3
)

// Profile は、Downstream ごとの resilient な振る舞いの設定です（infra 内部・port に漏らしません）。
type Profile struct {
	// PerAttemptTimeout は、1 回の試行(attempt)あたりのタイムアウトです。
	PerAttemptTimeout time.Duration
	// OverallTimeout は、retry を含む呼び出し全体のタイムアウトです。
	OverallTimeout time.Duration
	// MaxAttempts は、最大試行回数です（初回 + retry の合計）。
	MaxAttempts int
	// BaseBackoff は、指数バックオフの基準待機時間です。
	BaseBackoff time.Duration
	// MaxBackoff は、バックオフ待機時間の上限です。
	MaxBackoff time.Duration
	// RetryBudgetRatio は、リクエスト数に対する retry 許容比率です（per-downstream）。
	RetryBudgetRatio float64
	// MaxResponseBytes は、読み込むレスポンスボディの上限バイト数です。
	MaxResponseBytes int64
	// PropagateTrace は、この Downstream へ traceparent/baggage を注入するかを表します。
	// 信頼できない外部サービスへは false にし、内部相関 ID の外部漏洩を防ぎます。
	PropagateTrace bool
	// Breaker は、circuit breaker の設定です。
	Breaker BreakerConfig
}

// BreakerConfig は、circuit breaker の設定です。
type BreakerConfig struct {
	// FailureThreshold は、open に倒す失敗率です（0.0〜1.0）。
	FailureThreshold float64
	// MinRequests は、評価に必要な最小サンプル数です。
	MinRequests int
	// OpenDuration は、open から half-open へ遷移するまでの待機時間です。
	OpenDuration time.Duration
	// HalfOpenProbes は、half-open で通すプローブ数です。
	HalfOpenProbes int
}

// Registry は、Downstream から Profile を解決します。未登録キーは安全側のデフォルトに fallback します。
type Registry interface {
	// Profile は、d に対応する Profile を返します。未登録なら DefaultProfile を返します。
	Profile(d Downstream) Profile
}

// staticRegistry は、固定の Profile マップで構成された Registry です。
type staticRegistry struct {
	profiles map[Downstream]Profile
	fallback Profile
}

// DefaultProfile は、未登録 Downstream に適用される安全側のデフォルト Profile を返します。
func DefaultProfile() Profile {
	return Profile{
		PerAttemptTimeout: defaultPerAttemptTimeout,
		OverallTimeout:    defaultOverallTimeout,
		MaxAttempts:       defaultMaxAttempts,
		BaseBackoff:       defaultBaseBackoff,
		MaxBackoff:        defaultMaxBackoff,
		RetryBudgetRatio:  defaultRetryBudgetRatio,
		MaxResponseBytes:  defaultMaxResponseBytes,
		PropagateTrace:    defaultPropagateTrace,
		Breaker: BreakerConfig{
			FailureThreshold: defaultBreakerFailureThreshold,
			MinRequests:      defaultBreakerMinRequests,
			OpenDuration:     defaultBreakerOpenDuration,
			HalfOpenProbes:   defaultBreakerHalfOpenProbes,
		},
	}
}

// NewRegistry は、profiles を保持する Registry を生成します。未登録キーは DefaultProfile に fallback します。
func NewRegistry(profiles map[Downstream]Profile) Registry {
	return &staticRegistry{
		profiles: profiles,
		fallback: DefaultProfile(),
	}
}

// NewDefaultRegistry は、デフォルト Profile のみを持つ Registry を返します。
// 未登録の Downstream には DefaultProfile が適用されます。
func NewDefaultRegistry() Registry {
	return NewRegistry(nil)
}

// Profile は、d に対応する Profile を返します。未登録なら fallback を返します。
func (r *staticRegistry) Profile(d Downstream) Profile {
	if p, ok := r.profiles[d]; ok {
		return p
	}
	return r.fallback
}
