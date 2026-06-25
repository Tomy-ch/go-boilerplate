//go:generate mockgen -source=$GOFILE -destination=mock/mock_logger.gen.go -package=mock_$GOPACKAGE

// Package logging は、アプリケーションのロギング機能を提供します。
package logging

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LogCore は、Logger に追加で Tee できるログ core の型です。
type LogCore = zapcore.Core

// Logger は、アプリ全体が使うロガーのインターフェースです。
type Logger interface {
	// Debug はデバッグレベルのログを出力する。
	Debug(msg string, fields ...*Field)
	// Info は情報レベルのログを出力する。
	Info(msg string, fields ...*Field)
	// Warn は警告レベルのログを出力する。
	Warn(msg string, fields ...*Field)
	// Error はエラーレベルのログを出力する。
	Error(msg string, fields ...*Field)
	// Named は、新しい名前付きの Logger を返す。
	Named(name string) Logger
	// CallerSkip は、コールスタックのスキップ数を設定した新しいLoggerを返す。
	CallerSkip(skip int) Logger
}

type logger struct {
	log *zap.Logger
}

// levelGatedCore は、埋め込み core を最小レベル min で絞り込む zapcore.Core ラッパーです。
type levelGatedCore struct {
	zapcore.Core

	min zapcore.Level
}

// WithCore は、既存 Logger に追加のログ core を、元 Logger と同じ最小レベルでゲートして
// Tee した新しい Logger を返します。core が nil の場合は、元の Logger をそのまま返します。
func WithCore(l Logger, core LogCore) Logger {
	if core == nil {
		return l
	}
	// 本パッケージの具象 *logger だけが zap core を内包するため、Tee 対象も *logger に限定する。
	// それ以外の Logger 実装（テスト用 fake 等）は core を持たないので、ゲートできず素通しする。
	base, ok := l.(*logger)
	if !ok {
		return l
	}
	gated := levelGatedCore{Core: core, min: base.log.Level()}

	return &logger{
		log: base.log.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
			return zapcore.NewTee(c, gated)
		})),
	}
}

// Enabled は、min 以上かつ埋め込み core が有効なレベルのみ true を返します。
func (c levelGatedCore) Enabled(level zapcore.Level) bool {
	return level >= c.min && c.Core.Enabled(level)
}

// Check は、レベルが有効なときのみ自身を CheckedEntry に追加します。
func (c levelGatedCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

// With は、フィールドを付与しつつゲートを維持した core を返します。
func (c levelGatedCore) With(fields []zapcore.Field) zapcore.Core {
	return levelGatedCore{Core: c.Core.With(fields), min: c.min}
}

// Named は、新しい名前付きの Logger を返す。
func (l *logger) Named(name string) Logger {
	return &logger{
		log: l.log.Named(name),
	}
}

// CallerSkip は、コールスタックのスキップ数を設定した新しいLoggerを返す。
func (l *logger) CallerSkip(skip int) Logger {
	return &logger{
		log: l.log.WithOptions(zap.AddCallerSkip(skip)),
	}
}

// Debug はデバッグレベルのログを出力する。
func (l *logger) Debug(msg string, fields ...*Field) {
	l.log.Debug(msg, l.convertFields(fields)...)
}

// Info は情報レベルのログを出力する。
func (l *logger) Info(msg string, fields ...*Field) {
	l.log.Info(msg, l.convertFields(fields)...)
}

// Warn は警告レベルのログを出力する。
func (l *logger) Warn(msg string, fields ...*Field) {
	l.log.Warn(msg, l.convertFields(fields)...)
}

// Error はエラーレベルのログを出力する。
func (l *logger) Error(msg string, fields ...*Field) {
	l.log.Error(msg, l.convertFields(fields)...)
}

// convertFields は Field のスライスを zap.Field のスライスに変換する。
func (l *logger) convertFields(fs []*Field) []zap.Field {
	zfs := make([]zap.Field, 0, len(fs))

	for _, f := range fs {
		switch f.kind {
		case fieldString:
			zfs = append(zfs, zap.String(f.key, f.stringValue))
		case fieldStrings:
			zfs = append(zfs, zap.Strings(f.key, f.stringsValue))
		case fieldInt:
			zfs = append(zfs, zap.Int(f.key, f.intValue))
		case fieldInt64:
			zfs = append(zfs, zap.Int64(f.key, f.int64Value))
		case fieldFloat64:
			zfs = append(zfs, zap.Float64(f.key, f.float64Value))
		case fieldBool:
			zfs = append(zfs, zap.Bool(f.key, f.boolValue))
		case fieldError:
			zfs = append(zfs, zap.NamedError(f.key, f.errorValue))
		case fieldAny:
			zfs = append(zfs, zap.Any(f.key, f.anyValue))
		default:
			// fieldUnknown（ゼロ値）や将来追加され case 漏れした kind が
			// 到達する位置。バグ検知の余地を残しつつ zap.Any で安全に出力する。
			zfs = append(zfs, zap.Any(f.key, f.anyValue))
		}
	}

	return zfs
}
