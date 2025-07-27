// Package expectederrors は、NotFoundエラーの原因を定義します。
package expectederrors

type NotFoundCause string

const (
	NotFoundCauseDB      NotFoundCause = "db"
	NotFoundCauseCognito NotFoundCause = "cognito"
)

// IsDefinedNotFoundCause は、指定された NotFoundCause が定義済みの原因であるかどうかを判定します。
func IsDefinedNotFoundCause(cause NotFoundCause) bool {
	switch cause {
	case NotFoundCauseDB, NotFoundCauseCognito:
		return true
	default:
		return false
	}
}
