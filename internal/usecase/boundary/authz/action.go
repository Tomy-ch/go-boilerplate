package authz

const (
	// ActionUserGet は、ユーザー取得操作を表します。
	ActionUserGet Action = "user:get"
	// ActionUserUpdate は、ユーザー更新操作（全更新・部分更新）を表します。
	ActionUserUpdate Action = "user:update"
	// ActionUserDelete は、ユーザー削除操作を表します。
	ActionUserDelete Action = "user:delete"
	// ActionProductImageUpload は、商品画像のアップロード操作（admin）を表します。
	ActionProductImageUpload Action = "product:image:upload"
	// ActionProductCreate は、商品の作成操作（admin）を表します。
	ActionProductCreate Action = "product:create"
	// ActionProductUpdate は、商品の更新操作（admin）を表します。
	ActionProductUpdate Action = "product:update"
	// ActionProductStockUpdate は、商品在庫の増減操作（admin）を表します。
	ActionProductStockUpdate Action = "product:stock:update"
	// ActionProductListLowStock は、在庫僅少商品一覧の参照操作（admin）を表します。
	ActionProductListLowStock Action = "product:low-stock:list"
	// ActionPurchaseShip は、購入の発送操作（admin）を表します。
	ActionPurchaseShip Action = "purchase:ship"
	// ActionPurchaseDeliver は、購入の配達完了操作（admin）を表します。
	ActionPurchaseDeliver Action = "purchase:deliver"
	// ActionDashboardRead は、ダッシュボード集計の参照操作（admin）を表します。
	ActionDashboardRead Action = "dashboard:read"
)

// Action は、認可対象の操作を表す値オブジェクトです（例: "user:delete"）。
type Action string

// String は、Action の文字列表現を返します。
func (a Action) String() string { return string(a) }
