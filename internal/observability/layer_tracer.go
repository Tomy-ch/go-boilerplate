// Package observability は、可観測性に関連するユーティリティを提供します。
package observability

import (
	"context"
	"fmt"
	"time"

	"boilerplate-go/pkg/fnmeta"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type LayerTracer struct {
	log      *zap.Logger
	tracer   trace.Tracer
	layer    string
	pkgName  string
	funcName string
}

// Start は、新しい span を開始し、span を含む新しい Context と、span を終了するための endSpan 関数を返す。
//
// この時、Infoレベルでspanの開始・終了をログ出力する。
//
// 呼び出し元は処理完了時に endSpan を呼び出すことで span を確実に終了させることを想定している。
//
// 典型的な呼び出し方:
//
//	ctx, endSpan := layerTracer.Start(ctx)
//	defer endSpan()
func (lt LayerTracer) Start(
	ctx context.Context,
	opts ...trace.SpanStartOption,
) (context.Context, func()) {
	if lt.funcName == "" {
		full := getCallerFullName()
		lt.funcName = fnmeta.ExtractFunctionName(full)
	}

	return lt.spanStartBase(ctx, "", opts...)
}

// StartOptional は、新しい span を開始し、span を含む新しい Context と、span を終了するための endSpan 関数を返す。
//
// この時、Infoレベルでspanの開始・終了をログ出力する。
//
// optionalName を指定することで、span 名に追加情報を付与できる。
//
// 呼び出し元は処理完了時に endSpan を呼び出すことで span を確実に終了させることを想定している。
//
// 典型的な呼び出し方:
//
//	ctx, endSpan := layerTracer.StartOptional(ctx, "DBQuery")
//	defer endSpan()
func (lt LayerTracer) StartOptional(
	ctx context.Context,
	optionalName string,
	opts ...trace.SpanStartOption,
) (context.Context, func()) {
	if lt.funcName == "" {
		full := getCallerFullName()
		lt.funcName = fnmeta.ExtractFunctionName(full)
	}

	return lt.spanStartBase(ctx, optionalName, opts...)
}

// WithDomainSpan は、指定された関数 fn を新しい span 内で実行し、結果を返す。
//
// span の開始・終了時に Infoレベルでログ出力を行う。
//
// 呼び出し元は、関数 fn の実行結果とエラーを受け取ることができる。
//
// 典型的な呼び出し方:
//
//	ctx, result, err := layerTracer.WithDomainSpan(ctx, "user", "FullName", func(ctx context.Context) (T, error) {
//	    // 処理内容
//	})
func WithDomainSpan[T any](
	ctx context.Context,
	lt LayerTracer,
	pkg, funcName string,
	fn func(ctx context.Context) (T, error),
) (context.Context, T, error) {
	const layer = "domain"
	spanName := fmt.Sprintf("%s.%s.%s", layer, pkg, funcName)

	ctx, span := lt.tracer.Start(ctx, spanName)
	start := time.Now()

	lt.log.Info("span", lt.logFields(ctx, layer, pkg, funcName, "start", spanName)...)

	defer func() {
		duration := time.Since(start)
		span.End()
		lt.log.Info("span",
			lt.logFields(ctx, layer, pkg, funcName, "end", spanName,
				zap.Float64("duration_ms", float64(duration)/float64(time.Millisecond)),
			)...,
		)
	}()

	v, err := fn(ctx)
	if err != nil {
		var zero T
		return ctx, zero, err
	}

	return ctx, v, nil
}

// fullName は、LayerTracer の情報をもとに完全なspan名を生成します。
func (lt LayerTracer) fullName(optionalName string) string {
	fullName := lt.layer + "." + lt.pkgName + "." + lt.funcName
	if optionalName != "" {
		fullName += "." + optionalName
	}
	return fullName
}

// spanStartBase は、新しい span を開始し、span を含む新しい Context と、span を終了するための endSpan 関数を返す。
//
// optionalName を指定することで、span 名に追加情報を付与できる。
//
// このメソッドは LayerTracer の共通の span 開始ロジックを提供します。
//
// 呼び出し元は処理完了時に endSpan を呼び出すことで span を確実に終了させることを想定している。
//
// 典型的な呼び出し方:
//
//	ctx, endSpan := layerTracer.spanStartBase(ctx, "OptionalSpanName")
//	defer endSpan()
func (lt LayerTracer) spanStartBase(
	ctx context.Context,
	optionalName string,
	opts ...trace.SpanStartOption,
) (context.Context, func()) {
	spanName := lt.fullName(optionalName)
	ctx, span := lt.tracer.Start(ctx, spanName, opts...)

	startTime := time.Now()

	lt.log = lt.log.Named(spanName)
	lt.log.Info("span", lt.logFields(ctx, lt.layer, lt.pkgName, lt.funcName, "start", spanName)...)

	endSpan := func() {
		duration := time.Since(startTime)

		lt.log.Info("span", lt.logFields(ctx, lt.layer, lt.pkgName, lt.funcName, "end", spanName,
			zap.Float64("duration_ms", float64(duration)/float64(time.Millisecond)),
		)...)

		span.End()
	}

	return ctx, endSpan
}
