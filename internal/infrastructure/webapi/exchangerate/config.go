package exchangerate

// sampleEndpoint は、サンプル用の外部為替レートサービスのベース URL です。
const sampleEndpoint Endpoint = "https://api.exchangerate.example.com"

// NewEndpoint は、サンプル既定値の Endpoint を返します。
func NewEndpoint() Endpoint {
	return sampleEndpoint
}
