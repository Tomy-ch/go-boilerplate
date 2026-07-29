package observability

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metricnoop "go.opentelemetry.io/otel/metric/noop"

	"go-boilerplate/pkg/xerrors"
)

// newNoopMeterBuilder は、noop meter を持つ meterBuilder を返すテストヘルパーです。
func newNoopMeterBuilder(t *testing.T) *meterBuilder {
	t.Helper()
	return &meterBuilder{m: metricnoop.NewMeterProvider().Meter("test")}
}

func Test_meterBuilder_counter(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("エラー未発生なら counter を生成して返す", func(t *testing.T) {
			t.Parallel()

			b := newNoopMeterBuilder(t)

			c := b.counter("test.counter", "desc")

			assert.NotNil(t, c)
			require.NoError(t, b.err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("既にエラーがある場合は生成せず nil を返す", func(t *testing.T) {
			t.Parallel()

			b := newNoopMeterBuilder(t)
			b.err = xerrors.New("prior")

			c := b.counter("test.counter", "desc")

			assert.Nil(t, c)
		})
	})
}

func Test_meterBuilder_histogram(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("エラー未発生なら histogram を生成して返す", func(t *testing.T) {
			t.Parallel()

			b := newNoopMeterBuilder(t)

			h := b.histogram("test.histogram", "desc")

			assert.NotNil(t, h)
			require.NoError(t, b.err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("既にエラーがある場合は生成せず nil を返す", func(t *testing.T) {
			t.Parallel()

			b := newNoopMeterBuilder(t)
			b.err = xerrors.New("prior")

			h := b.histogram("test.histogram", "desc")

			assert.Nil(t, h)
		})
	})
}

func Test_meterBuilder_upDownCounter(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("エラー未発生なら upDownCounter を生成して返す", func(t *testing.T) {
			t.Parallel()

			b := newNoopMeterBuilder(t)

			c := b.upDownCounter("test.updown", "desc")

			assert.NotNil(t, c)
			require.NoError(t, b.err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("既にエラーがある場合は生成せず nil を返す", func(t *testing.T) {
			t.Parallel()

			b := newNoopMeterBuilder(t)
			b.err = xerrors.New("prior")

			c := b.upDownCounter("test.updown", "desc")

			assert.Nil(t, c)
		})
	})
}

func Test_meterBuilder_gauge(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("エラー未発生なら gauge を生成して返す", func(t *testing.T) {
			t.Parallel()

			b := newNoopMeterBuilder(t)

			g := b.gauge("test.gauge", "desc")

			assert.NotNil(t, g)
			require.NoError(t, b.err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("既にエラーがある場合は生成せず nil を返す", func(t *testing.T) {
			t.Parallel()

			b := newNoopMeterBuilder(t)
			b.err = xerrors.New("prior")

			g := b.gauge("test.gauge", "desc")

			assert.Nil(t, g)
		})
	})
}
