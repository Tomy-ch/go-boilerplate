package purchase

const (
	// StatusCodeUnprocessed は、購入作成直後に設定される「未処理」ステータスのコードです。
	// ステータスの UUID はドメインに焼き込まず、永続化時に code から解決します（seed との二重管理を避けるため）。
	StatusCodeUnprocessed = 1

	// StatusCodeCompleted は、購入完了ステータスのコードです。完了後はキャンセルできません（不正遷移 409）。
	StatusCodeCompleted = 5

	// StatusCodeCanceled は、購入キャンセルステータスのコードです。キャンセル遷移の遷移先です。
	StatusCodeCanceled = 6

	// StatusCodePaid は、購入支払い済みステータスのコードです。支払い遷移の遷移先で、未払い相当
	// （未処理 / 受付中 / 確認中 / 処理中）からのみ到達します。ステータスの UUID はドメインに焼き込まず、
	// 永続化時に code から解決します（seed との二重管理を避けるため）。
	// なお code 値（7）は安定した業務キーであり状態の到達順序を意味しない（完了=5・キャンセル=6 より大きい）。
	// 遷移判定は必ず等値比較で行い、大小比較（statusCode >= X 等）に用いてはならない。
	StatusCodePaid = 7

	// StatusCodeShipped は、購入発送済みステータスのコードです。発送遷移の遷移先で、支払い済みからのみ
	// 到達します。ステータスの UUID はドメインに焼き込まず、永続化時に code から解決します
	// （seed との二重管理を避けるため）。code 値（8）も到達順序を意味しないため、遷移判定は等値比較のみで行う。
	StatusCodeShipped = 8

	// StatusCodeDelivered は、購入配達済みステータスのコードです。発送済みからのみ到達し、発送・支払い・
	// キャンセルはいずれも不可能な終端状態です。配達遷移そのものは未実装ですが、購入ステータスマスタに
	// 存在する状態であり、他の遷移が「配達済みからは遷移できない」ことを表明するために必要です。
	StatusCodeDelivered = 9

	// taxRatePercent は、国内消費税率（パーセント）です。sample の placeholder であり、
	// 要件化した時点で config / マスタへ移します（ADR-0100）。
	taxRatePercent = 10

	// shippingFeeCents は、固定送料（USD セント）です。sample の placeholder です（ADR-0100）。
	shippingFeeCents = 500

	// percentDivisor は、パーセント計算の除数です（taxRatePercent を百分率として扱うため）。
	percentDivisor = 100

	// minorUnitDigits は、決済通貨（USD）の最小単位の小数桁数です（セント = 小数 2 桁）。
	// 価格スケール（ドル decimal）から決済スケール（整数セント）へ切り捨てる際の桁数に用います。
	minorUnitDigits = 2

	// minQuantity は、明細 1 件あたりの最小購入数量です。
	minQuantity = 1
)

// TerminalStatusCodes は、購入がそこから他の状態へ遷移しない終端ステータスのコードを返します。
// 終端でないステータスの購入は進行中として扱います。
func TerminalStatusCodes() []int {
	return []int{StatusCodeCompleted, StatusCodeCanceled, StatusCodeDelivered}
}
