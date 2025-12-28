//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE -package=mock_$GOPACKAGE

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

	// callerSkip は、ロガーのコールスタックのスキップ数を定義します。
	callerSkip = 3
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
func (lt LayerTracer) Start(
	ctx context.Context,
	opts ...trace.SpanStartOption,
) (context.Context, func()) {
	if lt.funcName == "" {
		full := getCallerFullName()
		lt.funcName = fnmeta.ExtractFunctionName(full)
	}

	return lt.startSpan(ctx, "", opts...)
}

// StartWithSuffix は、新しい span を開始し、span を含む新しい Context と、span を終了するための endSpan 関数を返す。
func (lt LayerTracer) StartWithSuffix(
	ctx context.Context,
	optionalName string,
	opts ...trace.SpanStartOption,
) (context.Context, func()) {
	if lt.funcName == "" {
		full := getCallerFullName()
		lt.funcName = fnmeta.ExtractFunctionName(full)
	}

	return lt.startSpan(ctx, optionalName, opts...)
}

// RunDomainWithSpan は、指定された関数 fn を新しい span 内で実行し、結果を返す。
//
// span の開始・終了時に Infoレベルでログ出力を行う。
//
// 呼び出し元は、関数 fn の実行結果とエラーを受け取ることができる。
//
// 典型的な呼び出し方:
//
//	ctx, result, err := layerTracer.RunDomainWithSpan(ctx, "user", "FullName", func(ctx context.Context) (T, error) {
//	    // 処理内容
//	})
func RunDomainWithSpan[T any](
	parentCtx context.Context,
	lt LayerTracer,
	pkg, funcName string,
	fn func(ctx context.Context) (T, error),
) (context.Context, T, error) {
	const layer = "domain"
	spanName := fmt.Sprintf("%s.%s.%s", layer, pkg, funcName)

	tc, childCtx, end := StartSpanWithParent(parentCtx, lt, spanName)
	start := time.Now()

	obsIn := logging.ObservabilityFieldsInput{
		SpanName:     spanName,
		Layer:        layer,
		PkgName:      pkg,
		FuncName:     funcName,
		TraceID:      tc.TraceID(),
		ParentSpanID: tc.ParentSpanID(),
		SpanID:       tc.SpanID(),
	}
	obsIn.EventType = SpanEventStart

	lt.logSpanEvent(callerSkip, obsIn)

	defer func() {
		event := SpanEventEnd
		obsIn.EventType = event
		obsIn.Latency = time.Since(start)
		lt.logSpanEvent(callerSkip+1, obsIn)
		end()
	}()

	v, err := fn(childCtx)
	if err != nil {
		var zero T
		return childCtx, zero, err
	}

	return childCtx, v, nil
}

// makeSpanName は、LayerTracer の情報をもとに完全なspan名を生成します。
func (lt LayerTracer) makeSpanName(optionalName string) string {
	fullName := lt.layer + delimiter + lt.pkgName + delimiter + lt.funcName
	if optionalName != "" {
		fullName += delimiter + optionalName
	}
	return fullName
}

// startSpan は、新しい span を開始し、span を含む新しい Context と、span を終了するための endSpan 関数を返す。
//
// optionalName を指定することで、span 名に追加情報を付与できる。
//
// このメソッドは LayerTracer の共通の span 開始ロジックを提供します。
//
// 呼び出し元は処理完了時に endSpan を呼び出すことで span を確実に終了させることを想定している。
func (lt LayerTracer) startSpan(
	parentCtx context.Context,
	optionalName string,
	opts ...trace.SpanStartOption,
) (context.Context, func()) {
	spanName := lt.makeSpanName(optionalName)
	tc, childCtx, end := StartSpanWithParent(parentCtx, lt, spanName, opts...)

	startTime := time.Now()

	obsIn := logging.ObservabilityFieldsInput{
		SpanName:     spanName,
		Layer:        lt.layer,
		PkgName:      lt.pkgName,
		FuncName:     lt.funcName,
		EventAt:      time.Now(),
		TraceID:      tc.TraceID(),
		SpanID:       tc.SpanID(),
		ParentSpanID: tc.ParentSpanID(),
	}

	obsIn.EventType = SpanEventStart
	lt.logSpanEvent(callerSkip+1, obsIn)

	endSpan := func() {
		obsIn.EventType = SpanEventEnd
		obsIn.Latency = time.Since(startTime)

		lt.logSpanEvent(callerSkip, obsIn)

		end()
	}

	return childCtx, endSpan
}

// logSpanEvent は、指定された観測可能性フィールドを使用してログを記録します。
func (lt LayerTracer) logSpanEvent(callerSkip int, obs logging.ObservabilityFieldsInput) {
	lt.log.Named(obs.Layer).CallerSkip(callerSkip).Info(
		obs.SpanName+delimiter+obs.EventType,
		lt.lf.BuildObservabilityFields(obs)...,
	)
}
