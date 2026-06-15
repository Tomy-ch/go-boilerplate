// Package xerrors は、エラーハンドリングのユーティリティを提供します。
package xerrors

//go:generate mockgen -source=$GOFILE -destination=mock/mock_interface.gen.go -package=mock_$GOPACKAGE

// Errors は、cockroachdb/errors への依存をこの抽象に閉じ込め差し替え可能に保つ契約です。
type Errors interface {
	// New は、新しいエラーを作成します。
	New(msg string) error
	// Wrap は、既存のエラーをラップして新しいエラーを作成します。
	Wrap(err error, msg string) error
	// Is は、エラーが特定のターゲットエラーと一致するかどうかを判定します。
	Is(err, target error) bool
	// As は、エラーが特定のターゲット型に変換可能かどうかを判定します。
	As(err error, target any) bool
	// StackTrace は、エラーの詳細表現（メッセージとスタックトレースを含む）を文字列で返します。
	StackTrace(err error) string
}
