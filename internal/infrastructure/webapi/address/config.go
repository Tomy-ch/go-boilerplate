package address

// sampleEndpoint は、サンプル用の住所検索サービスのベース URL です。
// 実在する公開サービスであり環境ごとに向き先が変わらないため、設定値ではなく定数で持ちます。
const sampleEndpoint Endpoint = "https://zipcloud.ibsnet.co.jp"

// NewEndpoint は、サンプル既定値の Endpoint を返します。
func NewEndpoint() Endpoint {
	return sampleEndpoint
}
