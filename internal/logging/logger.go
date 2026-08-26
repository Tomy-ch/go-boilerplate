//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package logging は、zap を基盤とする構造化ロギングの抽象を提供します。呼び出し側は zap へ直接
// 依存せず Logger 経由でログを出力し、ctx からの trace_id / span_id の自動注入と、HTTP / SQL の
// ログフィールド生成（LogFieldBuilder）を利用できます。
package logging

import (
	"context"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// traceFieldCount は、injectTrace が先頭へ注入する trace_id / span_id の 2 フィールド分。
// 注入後スライスの容量見積もりに使う。
const traceFieldCount = 2

// LogCore は、Logger に追加で Tee できるログ core の型です。
type LogCore = zapcore.Core

// TraceExtractor は、ctx から trace_id / span_id を抽出する関数です。ok=false のとき
// trace 情報を注入しません（logging を trace 実装から独立させ、循環参照を避けるための抽象）。
type TraceExtractor func(ctx context.Context) (traceID, spanID string, ok bool)

// Logger は、アプリ全体が使うロガーのインターフェースです。
// 各出力メソッドは ctx を受け取り、TraceExtractor が設定されていれば ctx 上の span から
// trace_id / span_id を注入します（未設定・span 無効時は注入しません）。
type Logger interface {
	// Debug はデバッグレベルのログを出力する。
	Debug(ctx context.Context, msg string, fields ...*Field)
	// Info は情報レベルのログを出力する。
	Info(ctx context.Context, msg string, fields ...*Field)
	// Warn は警告レベルのログを出力する。
	Warn(ctx context.Context, msg string, fields ...*Field)
	// Error はエラーレベルのログを出力する。
	Error(ctx context.Context, msg string, fields ...*Field)
	// Named は、新しい名前付きの Logger を返す。
	Named(name string) Logger
	// CallerSkip は、既存のスキップ数へ skip を加算した新しい Logger を返す。
	// 絶対値の設定ではないため、CallerSkip 済みの Logger へ重ねて呼ぶと積み上がる。
	CallerSkip(skip int) Logger
}

type logger struct {
	log     *zap.Logger
	extract TraceExtractor
}

// levelGatedCore は、埋め込み core を最小レベル min で絞り込む zapcore.Core ラッパーです。
type levelGatedCore struct {
	zapcore.Core

	min zapcore.Level
}

// WithCore は、既存 Logger に追加のログ core を、元 Logger と同じ最小レベルでゲートして
// Tee した新しい Logger を返します。core が nil の場合、および l が本パッケージの具象型でない
// 場合（テスト用 fake 等）は Tee できないため、元の Logger をそのまま返します。
func WithCore(l Logger, core LogCore) Logger {
	if core == nil {
		return l
	}
	base, ok := l.(*logger)
	if !ok {
		return l
	}
	gated := levelGatedCore{Core: core, min: base.log.Level()}

	return &logger{
		log: base.log.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
			return zapcore.NewTee(c, gated)
		})),
		extract: base.extract,
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

func (l *logger) Named(name string) Logger {
	return &logger{
		log:     l.log.Named(name),
		extract: l.extract,
	}
}

func (l *logger) CallerSkip(skip int) Logger {
	return &logger{
		log:     l.log.WithOptions(zap.AddCallerSkip(skip)),
		extract: l.extract,
	}
}

func (l *logger) Debug(ctx context.Context, msg string, fields ...*Field) {
	if ce := l.log.Check(zapcore.DebugLevel, msg); ce != nil {
		ce.Write(l.convertFields(l.injectTrace(ctx, fields))...)
	}
}

func (l *logger) Info(ctx context.Context, msg string, fields ...*Field) {
	if ce := l.log.Check(zapcore.InfoLevel, msg); ce != nil {
		ce.Write(l.convertFields(l.injectTrace(ctx, fields))...)
	}
}

func (l *logger) Warn(ctx context.Context, msg string, fields ...*Field) {
	if ce := l.log.Check(zapcore.WarnLevel, msg); ce != nil {
		ce.Write(l.convertFields(l.injectTrace(ctx, fields))...)
	}
}

func (l *logger) Error(ctx context.Context, msg string, fields ...*Field) {
	if ce := l.log.Check(zapcore.ErrorLevel, msg); ce != nil {
		ce.Write(l.convertFields(l.injectTrace(ctx, fields))...)
	}
}

// injectTrace は、TraceExtractor が trace を返す場合に trace_id / span_id を
// フィールド先頭へ注入する。extractor 未注入（nil）や span 無効時は fields をそのまま返す。
func (l *logger) injectTrace(ctx context.Context, fields []*Field) []*Field {
	if l.extract == nil {
		return fields
	}
	traceID, spanID, ok := l.extract(ctx)
	if !ok {
		return fields
	}

	out := make([]*Field, 0, len(fields)+traceFieldCount)
	out = append(out, String(TraceIDKey, traceID), String(SpanIDKey, spanID))
	return append(out, fields...)
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
