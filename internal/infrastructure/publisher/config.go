package publisher

import "go-boilerplate/internal/config"

// NewEndpoint は、config からメッセージ送信先エンドポイント URL を解決して返します。
func NewEndpoint(cfg *config.OutboxConfig) Endpoint {
	return Endpoint(cfg.Endpoint())
}
