package exchangerate

import (
	"context"
	"sync"
	"time"

	"go-boilerplate/internal/usecase/boundary/clock"
	boundary "go-boilerplate/internal/usecase/boundary/exchangerate"
)

// rateTTL は、レートキャッシュの有効期間です。外部レート源（ECB / Frankfurter）が日次更新のため
// 日次で十分であり、config 化しません。
const rateTTL = 24 * time.Hour

// maxCacheEntries は、キャッシュ保持数の上限です。エンドポイントは認証不要かつ base/quote が
// 自由入力の通貨コードのため、上限を設けないと map が無制限に成長しメモリ枯渇を招きうる。
// 上限到達時は失効エントリを掃除し、なお満杯なら新規保存を諦めます（取得済みレートは呼び出し元へ返る）。
const maxCacheEntries = 512

// cacheKey は、通貨ペアを一意に表すキーです。文字列連結だと区切り文字が値に混入した際に
// 別ペアと衝突しうるため、base / quote を分離した構造体で保持します。
type cacheKey struct {
	base  string
	quote string
}

// cacheEntry は、GetRate のキャッシュ 1 件分です。expiresAt を過ぎたら失効として再取得します。
type cacheEntry struct {
	rate      *boundary.Rate
	expiresAt time.Time
}

// cacheGateway は、boundary.Gateway を包む TTL キャッシュ decorator です。
// Echo はリクエスト毎に goroutine を割り当てるため、内部状態は sync.RWMutex で並行安全にします。
type cacheGateway struct {
	inner boundary.Gateway
	clk   clock.Clock
	mu    sync.RWMutex
	rates map[cacheKey]cacheEntry
}

// NewCache は、gateway を TTL キャッシュで包んだ boundary.Gateway を返します。
// clk は失効判定の時刻源で、テストでは固定時刻の clock を注入できます。
func NewCache(inner boundary.Gateway, clk clock.Clock) boundary.Gateway {
	return &cacheGateway{
		inner: inner,
		clk:   clk,
		rates: make(map[cacheKey]cacheEntry),
	}
}

// GetRate は、TTL 内ならキャッシュを返し、失効・未取得なら inner gateway から取得して保存します。
// エラーはキャッシュしません。
func (c *cacheGateway) GetRate(ctx context.Context, base, quote string) (*boundary.Rate, error) {
	key := cacheKey{base: base, quote: quote}

	c.mu.RLock()
	entry, ok := c.rates[key]
	c.mu.RUnlock()
	if ok && c.clk.Now().Before(entry.expiresAt) {
		return entry.rate, nil
	}

	rate, err := c.inner.GetRate(ctx, base, quote)
	if err != nil {
		return nil, err
	}

	c.store(key, rate)
	return rate, nil
}

// store は、レートを失効時刻付きで保存します。上限到達時はまず失効エントリを掃除し、
// なお満杯なら新規保存を諦めます。
func (c *cacheGateway) store(key cacheKey, rate *boundary.Rate) {
	now := c.clk.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.rates[key]; !exists && len(c.rates) >= maxCacheEntries {
		for k, e := range c.rates {
			if !now.Before(e.expiresAt) {
				delete(c.rates, k)
			}
		}
		if len(c.rates) >= maxCacheEntries {
			return
		}
	}
	c.rates[key] = cacheEntry{rate: rate, expiresAt: now.Add(rateTTL)}
}
