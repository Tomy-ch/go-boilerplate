package logging

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// stacktraceArrayCore は zap が自動付与する stacktrace（埋め込み \n\t の単一文字列）を、
// 行ごとの []string に変換して emit する zapcore.Core ラッパです。
// Grafana / Loki などの JSON ビューアでは配列要素ごとに改行表示されるため、
// 単一文字列のときに発生する \n のリテラル表示問題を避けられます。
type stacktraceArrayCore struct {
	zapcore.Core
	key string
}

// wrapStacktraceCore は、与えられた Core を stacktraceArrayCore でラップします。
// key は出力先キー名（EncoderConfig.StacktraceKey と一致させる）。
func wrapStacktraceCore(core zapcore.Core, key string) zapcore.Core {
	return &stacktraceArrayCore{Core: core, key: key}
}

// With は内側 Core の With を伝播しつつ、自身のラップ属性を維持します。
func (c *stacktraceArrayCore) With(fs []zapcore.Field) zapcore.Core {
	return &stacktraceArrayCore{Core: c.Core.With(fs), key: c.key}
}

// Check は、レベルが有効な場合に自身を CheckedEntry に登録します。
func (c *stacktraceArrayCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

// Write は、Entry.Stack を行配列に変換した zap.Strings フィールドへ差し替えます。
// 元の Entry.Stack はクリアして、エンコーダが二重に文字列出力するのを防ぎます。
func (c *stacktraceArrayCore) Write(ent zapcore.Entry, fs []zapcore.Field) error {
	if ent.Stack != "" {
		fs = append(fs, zap.Strings(c.key, SplitStackLines(ent.Stack)))
		ent.Stack = ""
	}
	return c.Core.Write(ent, fs)
}
