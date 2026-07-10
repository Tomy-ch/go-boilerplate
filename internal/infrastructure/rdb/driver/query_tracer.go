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
	recorder      QueryRecorder
	maskArgs      bool
	slowThreshold time.Duration
}

// NewQueryTracer は、span(otelpgx)・クエリログ(終了/スロー/エラー)・クエリメトリクス記録を行う pgx.QueryTracer を生成します。
func NewQueryTracer(
	dbCfg *config.DatabaseConfig,
	obsCfg *config.ObservabilityConfig,
	tracer *otelpgx.Tracer,
	recorder QueryRecorder,
	logger logging.Logger,
	lf logging.LogFieldBuilder,
) pgx.QueryTracer {
	return &queryTracer{
		Tracer:        tracer,
		logger:        logger.Named(queryTracerLayer),
		lf:            lf,
		recorder:      recorder,
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

	t.recordQueryMetric(ctx, ld.sql, duration, data.Err)

	switch {
	case data.Err != nil:
		t.logger.Error(ctx, "DB query failed", t.endFields(ld, duration, data.Err)...)
	case t.slowThreshold > 0 && duration > t.slowThreshold:
		t.logger.Warn(ctx, "DB slow query", t.endFields(ld, duration, nil)...)
	default:
		t.logger.Info(ctx, "DB query completed", t.endFields(ld, duration, nil)...)
	}
}

// recordQueryMetric は、recorder が設定されている場合にクエリメトリクスを記録します。
func (t *queryTracer) recordQueryMetric(ctx context.Context, sql string, duration time.Duration, err error) {
	if t.recorder == nil {
		return
	}
	t.recorder.Observe(ctx, buildQueryAttrs(ctx, sql, duration, err))
}

func (t *queryTracer) endFields(
	ld queryLogData, duration time.Duration, err error,
) []*logging.Field {
	args := ld.args
	if t.maskArgs {
		args = nil
	}

	return t.lf.BuildSQLEndFields(logging.SQLFieldsEndInput{
		Layer:        queryTracerLayer,
		PkgName:      queryTracerPkg,
		EventAt:      time.Now(),
		Latency:      duration,
		Query:        ld.sql,
		Args:         args,
		Err:          err,
		ParentSpanID: ld.parentSpanID,
	})
}
