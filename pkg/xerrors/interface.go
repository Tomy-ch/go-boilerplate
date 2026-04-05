// Package xerrors は、エラーハンドリングのユーティリティを提供します。
package xerrors

// Errors は、エラーハンドリングのインターフェースです。
//
//	スタックトレース機能を持つライブラリを使用して、エラーの生成とラップを行います。
type Errors interface {
	// New は、新しいエラーを作成します。
	New(msg string) error
	// Wrap は、既存のエラーをラップして新しいエラーを作成します。
	Wrap(err error, msg string) error
	// Is は、エラーが特定のターゲットエラーと一致するかどうかを判定します。
	Is(err, target error) bool
	// As は、エラーが特定のターゲット型に変換可能かどうかを判定します。
	As(err error, target any) bool
}
