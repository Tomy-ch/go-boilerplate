package logging

import "go.uber.org/zap/zapcore"

var (
	// LevelDebug は Debug レベルのログ指定子です。
	LevelDebug = Level{zapcore.DebugLevel}
	// LevelInfo は Info レベルのログ指定子です。
	LevelInfo = Level{zapcore.InfoLevel}
	// LevelWarn は Warn レベルのログ指定子です。
	LevelWarn = Level{zapcore.WarnLevel}
	// LevelError は Error レベルのログ指定子です。
	LevelError = Level{zapcore.ErrorLevel}
)

// Level は、ログ出力レベルを表す抽象型です。
// 保持します（ゼロ値は Info 相当）。
type Level struct {
	zl zapcore.Level
}

// ParseLevel は、"debug" / "info" / "warn" / "error" 等の文字列を Level に変換します。
// 解釈不能な文字列の場合はエラーを返します。
func ParseLevel(s string) (Level, error) {
	zl, err := zapcore.ParseLevel(s)
	if err != nil {
		return Level{}, err
	}
	return Level{zl: zl}, nil
}
