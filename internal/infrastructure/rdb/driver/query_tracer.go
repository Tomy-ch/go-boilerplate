package driver

import (
	"context"
	"time"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
)

const (
	queryTracerLayer = "infrastructure"
	queryTracerPkg   = "driver"
)

// queryLogKey は、クエリ開始情報を context に伝搬するためのキーです。
type queryLogKey struct{}

// queryLogData は、クエリ終了時のログ出力に必要な開始時点の情報です。
type queryLogData struct {
	sql          string
	args         []any
	start        time.Time
	parentSpanID string
}

// queryTracer は、span(otelpgx) に加え、正常終了(Info)・スロー(Warn)・エラー(Error)のログを
// 付加する pgx.QueryTracer の実装です。
type queryTracer struct {
	*otelpgx.Tracer

	logger        logging.Logger
	lf            logging.LogFieldBuilder
	maskArgs      bool
	slowThreshold time.Duration
}

// NewQueryTracer は、span(otelpgx) とクエリログ(終了/スロー/エラー)を行う pgx.QueryTracer を生成します。
func NewQueryTracer(
	dbCfg *config.DatabaseConfig,
	obsCfg *config.ObservabilityConfig,
	tracer *otelpgx.Tracer,
	logger logging.Logger,
	lf logging.LogFieldBuilder,
) pgx.QueryTracer {
	return &queryTracer{
		Tracer:        tracer,
		logger:        logger.Named(queryTracerLayer),
		lf:            lf,
		maskArgs:      obsCfg.MaskedDBQueryArgs(),
		slowThreshold: dbCfg.SlowQueryWarnThreshold(),
	}
}

// TraceQueryStart は、クエリ開始の span を開始し、新しい Context を返します。
func (t *queryTracer) TraceQueryStart(
	ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData,
) context.Context {
	parentSpanID := observability.ExtractTraceContext(ctx).SpanID()
	ctx = t.Tracer.TraceQueryStart(ctx, conn, data)
	return context.WithValue(ctx, queryLogKey{}, queryLogData{
		sql:          data.SQL,
		args:         data.Args,
		start:        time.Now(),
		parentSpanID: parentSpanID,
	})
}

// TraceQueryEnd は、span を終了し、正常終了(Info)・スロー(Warn)・エラー(Error)のログを出力します。
func (t *queryTracer) TraceQueryEnd(
	ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData,
) {
	t.Tracer.TraceQueryEnd(ctx, conn, data)

	ld, ok := ctx.Value(queryLogKey{}).(queryLogData)
	if !ok {
		return
	}
	duration := time.Since(ld.start)

	switch {
	case data.Err != nil:
		t.logger.Error("DB query failed", t.endFields(ctx, ld, duration, data.Err)...)
	case t.slowThreshold > 0 && duration > t.slowThreshold:
		t.logger.Warn("DB slow query", t.endFields(ctx, ld, duration, nil)...)
	default:
		t.logger.Info("DB query completed", t.endFields(ctx, ld, duration, nil)...)
	}
}

func (t *queryTracer) endFields(
	ctx context.Context, ld queryLogData, duration time.Duration, err error,
) []*logging.Field {
	args := ld.args
	if t.maskArgs {
		args = nil
	}

	tc := observability.ExtractTraceContext(ctx)

	return t.lf.BuildSQLEndFields(logging.SQLFieldsEndInput{
		Layer:        queryTracerLayer,
		PkgName:      queryTracerPkg,
		EventAt:      time.Now(),
		Latency:      duration,
		Query:        ld.sql,
		Args:         args,
		Err:          err,
		TraceID:      tc.TraceID(),
		SpanID:       tc.SpanID(),
		ParentSpanID: ld.parentSpanID,
	})
}
