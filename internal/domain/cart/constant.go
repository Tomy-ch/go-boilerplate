package cart

const (
	// minQuantity は、明細 1 件あたりの最小数量です。0 は削除ではなく不正な数量として扱います。
	minQuantity = 1

	// maxQuantityPerItem は、明細 1 件あたりの最大数量です。マージによる数量合算はこの値でクランプされます。
	maxQuantityPerItem = 99

	// maxItems は、1 カートが保持できる明細数の上限です。
	maxItems = 50

	// sessionTokenLength は、ゲストセッショントークンの長さです。
	// 256 ビットを base64url（パディング無し）で表現した長さに一致します。
	sessionTokenLength = 43
)
