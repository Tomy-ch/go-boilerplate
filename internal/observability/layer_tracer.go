//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package observability は、OTel ベースの分散トレース・メトリクス・ログ計装ユーティリティを提供します。
// LayerTracer によるアーキテクチャ層別 span、SSRF ガード付き HTTPClientTransport、
// Worker / HTTPClient / Outbox 向けメトリクスセット、OTLP Provider のセットアップ関数が含まれます。
package observability

import (
	"context"

	"go-boilerplate/pkg/fnmeta"

	"go.opentelemetry.io/otel/trace"
)

const (
	delimiter = "."
	// Controller は、コントローラー層を表すレイヤー名です。
	Controller layerName = "controller"
	// Usecase は、ユースケース層を表すレイヤー名です。
	Usecase layerName = "usecase"
	// Infra は、インフラ層を表すレイヤー名です。
	Infra layerName = "infrastructure"
)

type layerName string

// LayerTracer は、アーキテクチャレイヤー単位のトレース span を提供するトレーサーです。
type LayerTracer struct {
	tracer   trace.Tracer
	layer    layerName
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

// StartWithSuffix は、Start と同じく span を開始し、optionalName が空でなければ span 名の末尾に付与する。
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

// StartWithLink は、新しい span を開始し、carrier が指す trace への link を張ります。
// 親は ctx の span のままで、carrier は link にしかなりません — 起点の trace とは時も場所も
// 異なる作業を、その子にせず繋ぐためのものです。carrier が空か読めなければ link 無しの span になります。
func (lt LayerTracer) StartWithLink(
	ctx context.Context,
	carrier map[string]string,
	opts ...trace.SpanStartOption,
) (context.Context, func()) {
	if lt.funcName == "" {
		full := getCallerFullName()
		lt.funcName = fnmeta.ExtractFunctionName(full)
	}

	if link, ok := linkFromCarrier(carrier); ok {
		opts = append(opts, trace.WithLinks(link))
	}

	return lt.startSpan(ctx, "", opts...)
}

// linkFromCarrier は、carrier の trace context を span link に変えます。
func linkFromCarrier(carrier map[string]string) (trace.Link, bool) {
	sc := trace.SpanContextFromContext(
		extractFromCarrier(context.Background(), carrier, traceContextPropagator),
	)
	if !sc.IsValid() {
		return trace.Link{}, false
	}

	return trace.Link{SpanContext: sc}, true
}

// RunWithSpan は、指定された関数 fn を新しい span 内で実行し、結果を返す。
//
// 典型的な呼び出し方:
//
//	ctx, result, err := observability.RunWithSpan(ctx, lt, observability.Usecase, "user", "FullName", func(ctx context.Context) (T, error) {
//	    // 処理内容
//	})
func RunWithSpan[T any](
	parentCtx context.Context,
	lt LayerTracer,
	layer layerName,
	pkg, funcName string,
	fn func(ctx context.Context) (T, error),
) (context.Context, T, error) {
	spanName := BuildSpanName(string(layer), pkg, funcName)

	_, childCtx, end := StartSpanWithParent(parentCtx, lt, spanName)
	defer end()

	v, err := fn(childCtx)
	if err != nil {
		var zero T
		return childCtx, zero, err
	}

	return childCtx, v, nil
}

// makeSpanName は、LayerTracer の情報をもとに完全なspan名を生成します。
func (lt LayerTracer) makeSpanName(optionalName string) string {
	fullName := BuildSpanName(string(lt.layer), lt.pkgName, lt.funcName)
	if optionalName != "" {
		fullName += delimiter + optionalName
	}
	return fullName
}

// startSpan は、Start / StartWithSuffix / StartWithLink の共通実装で、makeSpanName(optionalName) の名前で span を開始する。
func (lt LayerTracer) startSpan(
	parentCtx context.Context,
	optionalName string,
	opts ...trace.SpanStartOption,
) (context.Context, func()) {
	spanName := lt.makeSpanName(optionalName)
	_, childCtx, end := StartSpanWithParent(parentCtx, lt, spanName, opts...)

	return childCtx, end
}
