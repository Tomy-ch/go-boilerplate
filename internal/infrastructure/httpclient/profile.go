package httpclient

import (
	"fmt"
	"time"

	"go-boilerplate/pkg/xerrors"
)

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
	// 既定では内部サービス利用を想定し private/loopback を許可（link-local は常に拒否）。
	// 外部 downstream は AllowPrivateNetwork=false を明示登録する。
	defaultAllowPrivateNetwork = true

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
	// AllowPrivateNetwork は、loopback / private(RFC1918, ULA) 宛て接続を許可するかを表します。
	// 内部サービス向けは true、信頼できない外部サービス向けは false にします。
	// link-local(クラウドメタデータ等)・unspecified は本フラグに関わらず常に拒否されます。
	AllowPrivateNetwork bool
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

// DownstreamProfile は、Downstream と Profile の組です。
type DownstreamProfile struct {
	// Name は、論理依存名です。
	Name Downstream
	// Profile は、その Downstream の resilient 設定です。
	Profile Profile
}

// staticRegistry は、固定の Profile マップで構成された Registry です。
type staticRegistry struct {
	profiles map[Downstream]Profile
	fallback Profile
}

// DefaultProfile は、未登録 Downstream に適用される安全側のデフォルト Profile を返します。
func DefaultProfile() Profile {
	return Profile{
		PerAttemptTimeout:   defaultPerAttemptTimeout,
		OverallTimeout:      defaultOverallTimeout,
		MaxAttempts:         defaultMaxAttempts,
		BaseBackoff:         defaultBaseBackoff,
		MaxBackoff:          defaultMaxBackoff,
		RetryBudgetRatio:    defaultRetryBudgetRatio,
		MaxResponseBytes:    defaultMaxResponseBytes,
		PropagateTrace:      defaultPropagateTrace,
		AllowPrivateNetwork: defaultAllowPrivateNetwork,
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

// NewRegistryFromProfiles は、各 gateway が寄与した DownstreamProfile 群から Registry を生成します。
// 未登録の Downstream には DefaultProfile が適用されます。
// 同一 Name が重複した場合は、silent な last-wins 上書きを避けるためエラーを返します。
func NewRegistryFromProfiles(profiles []DownstreamProfile) (Registry, error) {
	m := make(map[Downstream]Profile, len(profiles))
	for _, p := range profiles {
		if _, dup := m[p.Name]; dup {
			return nil, xerrors.New(fmt.Sprintf("duplicate httpclient profile for downstream %q", p.Name))
		}
		m[p.Name] = p.Profile
	}
	return NewRegistry(m), nil
}

// MissingDownstreams は、required のうち profiles に Profile が登録されていない Downstream を返します。
// 起動時の網羅チェックに使い、空スライスなら全ての required が登録済みであることを表します。
// これにより gateway 追加時の profile 登録漏れ・改名が、silent な DefaultProfile fallback ではなく
// loud な起動失敗として顕在化します。
func MissingDownstreams(profiles []DownstreamProfile, required []Downstream) []Downstream {
	registered := make(map[Downstream]struct{}, len(profiles))
	for _, p := range profiles {
		registered[p.Name] = struct{}{}
	}

	var missing []Downstream
	for _, d := range required {
		if _, ok := registered[d]; !ok {
			missing = append(missing, d)
		}
	}
	return missing
}

// Profile は、d に対応する Profile を返します。未登録なら fallback を返します。
func (r *staticRegistry) Profile(d Downstream) Profile {
	if p, ok := r.profiles[d]; ok {
		return p
	}
	return r.fallback
}
