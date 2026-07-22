package address

// sampleEndpoint は、サンプル用の外部郵便番号 lookup サービス（zipcloud）のベース URL です。
const sampleEndpoint Endpoint = "https://zipcloud.ibsnet.co.jp"

// NewEndpoint は、サンプル既定値の Endpoint を返します。
func NewEndpoint() Endpoint {
	return sampleEndpoint
}
