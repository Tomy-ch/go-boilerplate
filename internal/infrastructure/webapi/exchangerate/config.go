package exchangerate

import "go-boilerplate/internal/config"

// NewEndpoint は、設定から外部為替レートサービスの Endpoint を返します。
func NewEndpoint(epCfg *config.EndpointConfig) Endpoint {
	return Endpoint(epCfg.ExchangeRate())
}
