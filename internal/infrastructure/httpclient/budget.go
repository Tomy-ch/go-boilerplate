package httpclient

import "sync"

const (
	// retryBudgetMaxTokens は、per-downstream のトークン上限（バースト許容量）です。
	retryBudgetMaxTokens = 10.0
	// retryBudgetInitialTokens は、未知 Downstream の初期トークン残量です。
	// 初回満タンにすると ratio による retry 抑制が初期バーストを素通ししてしまうため、
	// 小さく始めてリクエストごとの refill で上限まで「獲得」させます。
	retryBudgetInitialTokens = 2.0
	// retryBudgetRetryCost は、retry 1 回あたりのトークン消費量です。
	retryBudgetRetryCost = 1.0
)

// retryBudget は、Downstream ごとの retry トークンバケットです。
// リクエストごとに ratio 分を補充し、retry ごとに 1 消費します。トークン枯渇時は retry を止めます。
type retryBudget struct {
	mu     sync.Mutex
	tokens map[Downstream]float64
}

// newRetryBudget は、retryBudget を生成します。
func newRetryBudget() *retryBudget {
	return &retryBudget{tokens: make(map[Downstream]float64)}
}

// refill は、リクエスト開始時に ratio 分のトークンを補充します（上限 retryBudgetMaxTokens）。
func (b *retryBudget) refill(d Downstream, ratio float64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	current, ok := b.tokens[d]
	if !ok {
		current = retryBudgetInitialTokens // 初回は小さく始め、refill で上限まで獲得させる
	}
	b.tokens[d] = min(retryBudgetMaxTokens, current+ratio)
}

// tryConsume は、retry 1 回分のトークンを消費できれば true を返します。
func (b *retryBudget) tryConsume(d Downstream) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.tokens[d] >= retryBudgetRetryCost {
		b.tokens[d] -= retryBudgetRetryCost
		return true
	}
	return false
}
