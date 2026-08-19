package address

import "go-boilerplate/internal/config"

// NewEndpoint は、設定から住所検索サービスの Endpoint を返します。
func NewEndpoint(epCfg *config.EndpointConfig) Endpoint {
	return Endpoint(epCfg.Address())
}
