// Package retry は、失敗分類を消費する有限リトライの行動層です。
//
// classify → bounded attempts → backoff + full jitter → deadline-aware という
// 共通ループを 1 度だけ実装し、tx リトライや外部 HTTP の retry が共有します。
// バックオフの待機時間は呼び出し側が [Policy.Backoff]（attempt → 基本待機時間）で
// 与え、本パッケージはそこへ full jitter（[Full]）を重畳します。乱数（math/rand/v2）
// 依存を本パッケージ側へ閉じることで、待機時間算出を担う純粋な pkg/backoff の
// 「現在時刻や乱数に依存しない」純粋性を保ちます。
//
// pkg は相互独立（pkg→pkg import 禁止）のため、Backoff は pkg/backoff の型ではなく
// 関数値として受け取ります。
package retry

import (
	"context"
	"time"
)

// Sleeper は、リトライ間の待機を抽象化するインターフェースです。
// ctx のキャンセル / deadline を尊重し、待機が打ち切られた場合は error を返します。
//
// 本パッケージは internal 非依存のため独自に定義します。Sleep(ctx context.Context, d time.Duration) error を持つ任意の型を注入できます。
type Sleeper interface {
	// Sleep は、d 経過まで待機します。ctx 打ち切り時は error を返します。
	Sleep(ctx context.Context, d time.Duration) error
}

// Policy は、有限リトライの設定です。
type Policy struct {
	// MaxAttempts は、fn の最大試行回数です。1 未満は 1 として扱います（最低 1 回試行）。
	MaxAttempts int
	// Backoff は、attempt（0 起算）に対する jitter 適用前の基本待機時間を返します。
	// nil の場合は待機 0（即時リトライ）。実待機は本値に [Full] を重畳したものです。
	// 呼び出し側は backoff.Exponential.Duration 等をそのまま渡せます。
	Backoff func(attempt int) time.Duration
}

// Do は、fn を有限リトライで実行します。
//
// fn の返した error が isRetryable を満たす限り、policy.MaxAttempts まで再試行し、
// 試行間は policy.Backoff（+ full jitter）だけ sleeper で待機します。最後に観測した
// error（成功時は nil）を返します。
//
//   - fn が nil を返した、isRetryable が false、または最終試行に達した場合は即座に返します。
//   - sleeper.Sleep が error を返した（ctx 打ち切り）場合は、sleep の error ではなく
//     直前の fn の error を返します（リトライ対象だった元の失敗を呼び出し側へ伝えるため）。
//
// isRetryable は fn が返した非 nil error に対してのみ呼ばれます。isRetryable 自体が nil の
// 場合は全ての error をリトライ不可として扱い、最初の error で即座に返します。
func Do(
	ctx context.Context,
	sleeper Sleeper,
	policy Policy,
	isRetryable func(error) bool,
	fn func(context.Context) error,
) error {
	if isRetryable == nil {
		isRetryable = func(error) bool { return false }
	}
	attempts := max(1, policy.MaxAttempts)

	var err error
	for attempt := 1; attempt <= attempts; attempt++ {
		err = fn(ctx)
		if err == nil || !isRetryable(err) || attempt == attempts {
			return err
		}

		var base time.Duration
		if policy.Backoff != nil {
			base = policy.Backoff(attempt - 1)
		}
		if serr := sleeper.Sleep(ctx, Full(base)); serr != nil {
			return err
		}
	}
	return err
}
