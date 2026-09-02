package stream

import (
	"time"

	ucrealtime "go-boilerplate/internal/usecase/realtime"
)

// Settings のゼロ値に充てる既定値。どちらも instance あたりの上限で、負荷と機材で決まるため
// 配備ごとに変わりえます（REALTIME_MAX_CONNECTIONS / REALTIME_REPLAY_CONCURRENCY）。
const (
	// DefaultMaxConnections は、1 instance が同時に保持する SSE 接続数の上限です。
	DefaultMaxConnections = 1000
	// DefaultReplayConcurrency は、replay と catch-up が同時に走る本数の上限です。
	DefaultReplayConcurrency = 16
)

// 接続の振る舞いを決める固定値。docs/design/realtime-delivery.md §3.3 の "Fixed in code" に対応し、
// ここが唯一の定義です。
const (
	// writeDeadline は、1 回の書き込みに与える猶予です。超えた接続は遅い client として閉じます。
	writeDeadline = 10 * time.Second
	// heartbeatInterval は、id を持たない comment を送る間隔です。到達しない peer をこれで検出します。
	heartbeatInterval = 15 * time.Second
	// catchUpInterval は、wakeup が届かなくても EventLog を読み直す間隔です。
	catchUpInterval = 30 * time.Second
	// catchUpJitter は、catchUpInterval に上乗せする揺らぎの幅です。全接続が同時刻に読みに行くのを防ぎます。
	catchUpJitter = catchUpInterval / 5
	// maxConnectionLifetime は、確立済み接続の寿命です。到達したら REAUTHENTICATE を送って閉じます。
	// ticket の TTL とは別物で、こちらは接続を、あちらは新規接続の開始を有界にします。
	maxConnectionLifetime = time.Hour
	// admissionWait は、初回 replay の枠が空くのを待つ上限です。超えたらレスポンス確定前に 503 を返します。
	admissionWait = 2 * time.Second
	// drainBudget は、停止時に接続を閉じ切るのを待つ上限です。停止 ctx の残りとこの値の短いほうを使い、
	// 残りを常駐処理の停止・instance resource の片付け・HTTP shutdown に残します。
	drainBudget = 10 * time.Second
	// retryAfterHint は、503 の Retry-After（秒）と RETRY_LATER の retryAfterMs に共用する目安です。
	retryAfterHint = 5 * time.Second
	// connectionBuffer は、1 接続が client への送出を待たせられる event 数です。1 ページが buffer を
	// 溢れさせないよう replay の 1 ページと同じ大きさにします。
	connectionBuffer = ucrealtime.ReplayPageLimit
)

// Settings は、Streamer のチューニング値です。ゼロ値は既定値に寄せます
// （consumer engine の realtime.Settings と同じ形）。
type Settings struct {
	// MaxConnections は、1 instance が同時に保持する SSE 接続数の上限です。
	MaxConnections int
	// ReplayConcurrency は、replay と catch-up が同時に走る本数の上限です。
	ReplayConcurrency int
}

// withDefaults は、未設定の値を既定値で埋めた Settings を返します。
func (s Settings) withDefaults() Settings {
	if s.MaxConnections <= 0 {
		s.MaxConnections = DefaultMaxConnections
	}

	if s.ReplayConcurrency <= 0 {
		s.ReplayConcurrency = DefaultReplayConcurrency
	}

	return s
}
