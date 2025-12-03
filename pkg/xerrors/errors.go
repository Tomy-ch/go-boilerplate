package xerrors

import (
	"fmt"

	"github.com/cockroachdb/errors"
)

// New は、新しいエラーを生成します。
func New(msg string) error {
	return errors.New(msg)
}

// Wrap は、既存のエラーをラップして新しいエラーを生成します。
// msg はラップする際の追加メッセージです。メッセージは元のエラーの前に付加されます。
func Wrap(err error, msg string) error {
	return errors.Wrap(err, msg)
}

// Is は、エラーが特定のターゲットエラーと一致するかを判定します。
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As は、エラーが特定のターゲット型に変換可能かを判定します。
func As(err error, target any) bool {
	return errors.As(err, target)
}

// StackTrace は、エラーのスタックトレースを文字列として取得します。
func StackTrace(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("%+v", err)
}
