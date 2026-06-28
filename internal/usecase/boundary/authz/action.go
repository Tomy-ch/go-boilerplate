package authz

const (
	// ActionUserGet は、ユーザー取得操作を表します。
	ActionUserGet Action = "user:get"
	// ActionUserUpdate は、ユーザー更新操作（全更新・部分更新）を表します。
	ActionUserUpdate Action = "user:update"
	// ActionUserDelete は、ユーザー削除操作を表します。
	ActionUserDelete Action = "user:delete"
)

// Action は、認可対象の操作を表す値オブジェクトです（例: "user:delete"）。
type Action string

// String は、Action の文字列表現を返します。
func (a Action) String() string { return string(a) }
