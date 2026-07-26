// Package xerrors は、StackTrace を含むエラーのラップ・判定・結合を提供します。
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
// err が nil の場合は nil を返します。
func Wrap(err error, msg string) error {
	return errors.Wrap(err, msg)
}

// Is は、err のエラーチェーン中に target と一致するエラーが含まれるかを判定します。
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As は、err のエラーチェーンから target が指す型に一致する最初のエラーを探し、見つかった場合は
// target へその値を設定して true を返します。target は非 nil のポインタである必要があります。
func As(err error, target any) bool {
	return errors.As(err, target)
}

// Join は、複数のエラーを結合して新しいエラーを生成します。
// nil の要素は捨てられ、非 nil が 1 つも無い場合は nil を返します。
func Join(errs ...error) error {
	return errors.Join(errs...)
}

// StackTrace は、err の詳細文字列表現を返します。err が cockroachdb/errors 等でスタックトレースを付加されている場合はそれも含みます。err が nil の場合は空文字列を返します。
func StackTrace(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("%+v", err)
}
