package logging

import (
	"fmt"

	"go-boilerplate/internal/config"
	"go-boilerplate/pkg/xerrors"

	"go.uber.org/zap/zapcore"
)

// Level は、ログ出力レベルを表す型です（ゼロ値は Info 相当）。
type Level struct {
	zl zapcore.Level
}

// LevelDebug は Debug レベルを返します。
func LevelDebug() Level { return Level{zapcore.DebugLevel} }

// LevelInfo は Info レベルを返します。
func LevelInfo() Level { return Level{zapcore.InfoLevel} }

// LevelWarn は Warn レベルを返します。
func LevelWarn() Level { return Level{zapcore.WarnLevel} }

// LevelError は Error レベルを返します。
func LevelError() Level { return Level{zapcore.ErrorLevel} }

// ParseLevel は、APP_LOG_LEVEL が受理するレベル文字列を Level に変換します。
// 受理しない文字列の場合はエラーを返します。
func ParseLevel(s string) (Level, error) {
	switch s {
	case config.LogLevelDebug:
		return LevelDebug(), nil
	case config.LogLevelInfo:
		return LevelInfo(), nil
	case config.LogLevelWarn:
		return LevelWarn(), nil
	case config.LogLevelError:
		return LevelError(), nil
	default:
		return Level{}, xerrors.New(fmt.Sprintf("unsupported log level: %q", s))
	}
}
