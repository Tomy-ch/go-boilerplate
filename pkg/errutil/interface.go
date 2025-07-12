package errUtil

// XError は、エラーハンドリングのインターフェースです。
//
//	スタックトレース機能を持つライブラリを使用して、エラーの生成とラップを行います。
type XErrors interface {
	// New は、新しいエラーを作成します。
	New(msg string) error
	// Wrap は、既存のエラーをラップして新しいエラーを作成します。
	Wrap(err error, msg string) error
	// Is は、エラーが特定のターゲットエラーと一致するかどうかを判定します。
	Is(err, target error) bool
}
