package status

const (
	minNameLength = 1
	maxNameLength = 100
	// minCode / maxCode および minSortKey / maxSortKey は、符号付き 16bit 整数の正数範囲です。
	// この範囲を業務値（purchase 集約の値オブジェクト Status が持つ code 集合）へ狭めないこと。
	// 詳細: docs/spec/purchase-status/domain.md
	minCode    = 1
	maxCode    = 32767
	minSortKey = 1
	maxSortKey = 32767
)
