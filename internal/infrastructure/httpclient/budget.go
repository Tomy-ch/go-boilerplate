package httpclient

import "sync"

const (
	// retryBudgetMaxTokens は、per-downstream のトークン上限（バースト許容量）です。
	retryBudgetMaxTokens = 10.0
	// retryBudgetRetryCost は、retry 1 回あたりのトークン消費量です。
	retryBudgetRetryCost = 1.0
)

// retryBudget は、per-downstream の retry トークンバケットです（A-5 / D-7）。
// リクエストごとに ratio 分を補充し、retry ごとに 1 消費します。連続失敗でトークンが枯渇すると
// retry が止まり fail-fast します（thundering retry を抑制）。
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
		current = retryBudgetMaxTokens // 初回は満タンから開始する
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
