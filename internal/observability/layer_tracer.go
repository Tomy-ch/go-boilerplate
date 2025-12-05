// Package observability は、可観測性に関連するユーティリティを提供します。
package observability

import (
	"context"
	"fmt"
	"time"

	"boilerplate-go/internal/logging"
	"boilerplate-go/pkg/fnmeta"

	"go.opentelemetry.io/otel/trace"
)

const (
	delimiter                = "."
	tracerNameController     = "controller"
	tracerNameUsecase        = "usecase"
	tracerNameInfrastructure = "infrastructure"

	// callerSkip は、zap ロガーのコールスタックのスキップ数を定義します。
	callerSkip = 1
)

type LayerTracer struct {
	log      logging.Logger
	lf       logging.LogFieldBuilder
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

	obsIn := logging.ObservabilityFieldsInput{
		SpanName: spanName,
		Layer:    layer,
		PkgName:  pkg,
		FuncName: funcName,
		TraceID:  span.SpanContext().TraceID().String(),
		SpanID:   span.SpanContext().SpanID().String(),
	}
	obsIn.SpanEvent = SpanEventStart

	lt.log.CallerSkip(callerSkip).Info(spanName+delimiter+SpanEventStart, lt.lf.BuildObservabilityFields(obsIn)...)

	defer func() {
		event := SpanEventEnd
		obsIn.SpanEvent = event
		obsIn.Latency = time.Since(start)
		lt.log.CallerSkip(callerSkip+1).Info(spanName+delimiter+event, lt.lf.BuildObservabilityFields(obsIn)...)
		span.End()
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

	obsIn := logging.ObservabilityFieldsInput{
		SpanName: spanName,
		Layer:    lt.layer,
		PkgName:  lt.pkgName,
		FuncName: lt.funcName,
		TraceID:  span.SpanContext().TraceID().String(),
		SpanID:   span.SpanContext().SpanID().String(),
	}

	obsIn.SpanEvent = SpanEventStart
	lt.log.CallerSkip(callerSkip+1).Info(spanName+delimiter+SpanEventStart, lt.lf.BuildObservabilityFields(obsIn)...)

	endSpan := func() {
		obsIn.SpanEvent = SpanEventEnd
		obsIn.Latency = time.Since(startTime)

		lt.log.CallerSkip(callerSkip).Info(spanName+delimiter+SpanEventEnd, lt.lf.BuildObservabilityFields(obsIn)...)

		span.End()
	}

	return ctx, endSpan
}
