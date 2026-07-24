package product

const (
	minNameLength = 1
	maxNameLength = 255
	minQuantity   = 0
	minThreshold  = 0
	// minRefNameLength / maxRefNameLength は、ステータス／カテゴリ参照が保持する名称の長さ範囲です。
	// 参照元マスタ（product_statuses / product_categories）の名称長制約に揃えています。
	minRefNameLength = 1
	maxRefNameLength = 100
)
