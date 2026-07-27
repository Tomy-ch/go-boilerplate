package observability

import (
	"go.opentelemetry.io/otel/metric"

	"go-boilerplate/pkg/xerrors"
)

// meterBuilder は、計装生成の最初のエラーを保持しつつ宣言的に組み立てます。
type meterBuilder struct {
	m   metric.Meter
	err error
}

func (b *meterBuilder) counter(name, desc string) metric.Int64Counter {
	if b.err != nil {
		return nil
	}
	c, err := b.m.Int64Counter(name, metric.WithDescription(desc))
	if err != nil {
		b.err = xerrors.Wrap(err, name)
		return nil
	}
	return c
}

func (b *meterBuilder) histogram(name, desc string) metric.Float64Histogram {
	if b.err != nil {
		return nil
	}
	h, err := b.m.Float64Histogram(name, metric.WithDescription(desc), metric.WithUnit("ms"))
	if err != nil {
		b.err = xerrors.Wrap(err, name)
		return nil
	}
	return h
}

func (b *meterBuilder) upDownCounter(name, desc string) metric.Int64UpDownCounter {
	if b.err != nil {
		return nil
	}
	c, err := b.m.Int64UpDownCounter(name, metric.WithDescription(desc))
	if err != nil {
		b.err = xerrors.Wrap(err, name)
		return nil
	}
	return c
}

func (b *meterBuilder) gauge(name, desc string) metric.Int64Gauge {
	if b.err != nil {
		return nil
	}
	g, err := b.m.Int64Gauge(name, metric.WithDescription(desc))
	if err != nil {
		b.err = xerrors.Wrap(err, name)
		return nil
	}
	return g
}
