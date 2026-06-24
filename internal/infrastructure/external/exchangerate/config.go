package exchangerate

// sampleEndpoint は、サンプル用の外部為替レートサービスのベース URL です。
// 実運用では config（環境変数）から注入する形へ差し替えます。
const sampleEndpoint Endpoint = "https://api.exchangerate.example.com"

// NewEndpoint は、DI 用に Endpoint を供給します（サンプル既定値）。
func NewEndpoint() Endpoint {
	return sampleEndpoint
}
