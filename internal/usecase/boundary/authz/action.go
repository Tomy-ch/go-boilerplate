package authz

// Action は、認可対象の操作を表す値オブジェクトです（例: "user:delete"）。
type Action string

// String は、Action の文字列表現を返します。
func (a Action) String() string { return string(a) }
