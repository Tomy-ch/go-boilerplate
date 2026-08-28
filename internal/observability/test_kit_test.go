package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestNewNoopTracerFactory(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Noop TracerProviderを保持するfactoryを返す", func(t *testing.T) {
			t.Parallel()
			tp := noop.NewTracerProvider()

			actual := NewNoopTracerFactory(t)
			tf, ok := actual.(*tracerFactory)
			require.True(t, ok)
			assert.Equal(t, tp, tf.tp)
		})
	})
}

func TestNewMockControllerLayerTracer(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Controllerレイヤを設定したMockLayerTracerを返す", func(t *testing.T) {
			t.Parallel()
			actual := NewMockControllerLayerTracer(t)
			assert.Equal(t, Controller, actual.layer)
			assert.NotNil(t, actual.tracer)
		})
	})
}

func TestNewMockUsecaseLayerTracer(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("UsecaseレイヤをセットしたMockLayerTracerを返す", func(t *testing.T) {
			t.Parallel()
			actual := NewMockUsecaseLayerTracer(t)
			assert.Equal(t, Usecase, actual.layer)
			assert.NotNil(t, actual.tracer)
		})
	})
}

func TestNewMockInfraLayerTracer(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("InfraレイヤをセットしたMockLayerTracerを返す", func(t *testing.T) {
			t.Parallel()
			actual := NewMockInfraLayerTracer(t)
			assert.Equal(t, Infra, actual.layer)
			assert.NotNil(t, actual.tracer)
		})
	})
}

func TestNewNoopLayerTracer(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("テスト用LayerTracerを返す", func(t *testing.T) {
			t.Parallel()
			actual := NewNoopLayerTracer(t)
			assert.Equal(t, layer, actual.layer)
			assert.Equal(t, pkg, actual.pkgName)
			assert.NotNil(t, actual.tracer)
		})
	})
}

func TestNewNoopWorkerMetrics(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("no-op MeterProvider から WorkerMetrics を生成する", func(t *testing.T) {
			t.Parallel()

			wm := NewNoopWorkerMetrics(t)
			require.NotNil(t, wm)
			assert.NotPanics(t, func() { wm.Processed(context.Background()) })
		})
	})
}

func TestNewNoopHTTPClientMetrics(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("no-op MeterProvider から HTTPClientMetrics を生成する", func(t *testing.T) {
			t.Parallel()

			hm := NewNoopHTTPClientMetrics(t)
			require.NotNil(t, hm)
			assert.NotPanics(t, func() { hm.RecordRetry(context.Background(), "sample") })
		})
	})
}

func TestNewNoopOutboxMetrics(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("no-op MeterProvider から OutboxMetrics を生成する", func(t *testing.T) {
			t.Parallel()

			om := NewNoopOutboxMetrics(t)
			assert.NotNil(t, om)
		})
	})
}

func TestNewNoopHTTPClientTransport(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("SSRFガードを無効化し loopback 宛ての実接続を許可する", func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}))
			defer srv.Close()

			transport := NewNoopHTTPClientTransport(t)
			require.NotNil(t, transport)

			// permissive な dial control により loopback(httptest) 宛ての実 dial が拒否されず接続できる。
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
			require.NoError(t, err)
			resp, err := (&http.Client{Transport: transport.RoundTripper()}).Do(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, http.StatusNoContent, resp.StatusCode)
		})
	})
}

func TestNewGuardedHTTPClientTransport(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("SSRFガードを残し方針未指定の loopback 宛てを拒否する", func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}))
			defer srv.Close()

			transport := NewGuardedHTTPClientTransport(t)
			require.NotNil(t, transport)

			// permissive でない dial control のため、許可を積んでいない loopback 宛ては dial で落ちる。
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
			require.NoError(t, err)
			_, err = (&http.Client{Transport: transport.RoundTripper()}).Do(req)

			require.Error(t, err)
		})
	})
}

func TestNewStubSpanContext(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("有効なSpanContextを保持するcontextとspan終了関数を返す", func(t *testing.T) {
			t.Parallel()
			ctx, span := NewStubSpanContext(t)
			require.NotEmpty(t, ctx)
			require.NotNil(t, span)
			defer span()

			spanCtx := trace.SpanFromContext(ctx)
			assert.True(t, spanCtx.SpanContext().IsValid())
		})
	})
}

func TestNewObservedHTTPClientMetrics(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("計上を読み出せる HTTPClientMetrics を返す", func(t *testing.T) {
			t.Parallel()

			obs := NewObservedHTTPClientMetrics(t)
			require.NotNil(t, obs.HTTPClientMetrics)

			obs.RecordError(context.Background(), "acct", "transport")

			assert.Equal(t, []string{"transport"}, obs.LabelValues(t, "httpclient.errors", "reason"))
		})
	})
}

func TestObservedHTTPClientMetrics_LabelValues(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("計上された label 組を全件返す", func(t *testing.T) {
			t.Parallel()

			obs := NewObservedHTTPClientMetrics(t)
			ctx := context.Background()
			obs.RecordError(ctx, "acct", "transport")
			obs.RecordError(ctx, "acct", "canceled")

			assert.ElementsMatch(t, []string{"transport", "canceled"}, obs.LabelValues(t, "httpclient.errors", "reason"))
		})

		t.Run("同一 label 組への複数回の記録は 1 件へ集約される", func(t *testing.T) {
			t.Parallel()

			obs := NewObservedHTTPClientMetrics(t)
			ctx := context.Background()
			obs.RecordError(ctx, "acct", "transport")
			obs.RecordError(ctx, "acct", "transport")

			assert.Equal(t, []string{"transport"}, obs.LabelValues(t, "httpclient.errors", "reason"))
		})

		t.Run("未計上の指標に対しては空を返す", func(t *testing.T) {
			t.Parallel()

			obs := NewObservedHTTPClientMetrics(t)

			assert.Empty(t, obs.LabelValues(t, "httpclient.errors", "reason"))
		})
	})
}

func Test_counterLabelValues(t *testing.T) {
	t.Parallel()

	// collect は、obs への計上結果を ResourceMetrics として取り出す。
	collect := func(t *testing.T, obs *ObservedHTTPClientMetrics) metricdata.ResourceMetrics {
		t.Helper()
		var rm metricdata.ResourceMetrics
		require.NoError(t, obs.reader.Collect(context.Background(), &rm))
		return rm
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定した counter のデータポイントから label 値を全件返す", func(t *testing.T) {
			t.Parallel()

			obs := NewObservedHTTPClientMetrics(t)
			ctx := context.Background()
			obs.RecordError(ctx, "acct", "transport")
			obs.RecordError(ctx, "acct", "canceled")

			got, ok := counterLabelValues(collect(t, obs), "httpclient.errors", "reason")

			require.True(t, ok)
			assert.ElementsMatch(t, []string{"transport", "canceled"}, got)
		})

		t.Run("指標に存在しない label キーは値を返さない", func(t *testing.T) {
			t.Parallel()

			obs := NewObservedHTTPClientMetrics(t)
			obs.RecordError(context.Background(), "acct", "transport")

			got, ok := counterLabelValues(collect(t, obs), "httpclient.errors", "unknown_label")

			require.True(t, ok)
			assert.Empty(t, got)
		})

		t.Run("名前が一致しない指標は無視する", func(t *testing.T) {
			t.Parallel()

			obs := NewObservedHTTPClientMetrics(t)
			obs.RecordError(context.Background(), "acct", "transport")

			got, ok := counterLabelValues(collect(t, obs), "httpclient.requests", "status_class")

			require.True(t, ok)
			assert.Empty(t, got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("int64 counter でない指標を指定した場合は ok=false を返す", func(t *testing.T) {
			t.Parallel()

			obs := NewObservedHTTPClientMetrics(t)
			// latency は Histogram のため counter として読み出せない（テストキットの誤用にあたる）。
			obs.RecordLatencyMs(context.Background(), "acct", 12)

			got, ok := counterLabelValues(collect(t, obs), "httpclient.request_latency_ms", "downstream")

			assert.False(t, ok)
			assert.Nil(t, got)
		})
	})
}

func TestNewRecordingTracerProvider(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("終了した span を名前と属性ごと保持する", func(t *testing.T) {
			t.Parallel()

			tp, recorded := NewRecordingTracerProvider(t)
			_, span := tp.Tracer("test").Start(context.Background(), "probe", trace.WithAttributes(attribute.String("k", "v")))
			span.End()

			spans := recorded()
			require.Len(t, spans, 1)
			assert.Equal(t, "probe", spans[0].Name())
			assert.Contains(t, spans[0].Attributes(), attribute.String("k", "v"))
		})

		t.Run("親 span へのリンクを保持する", func(t *testing.T) {
			t.Parallel()

			tp, recorded := NewRecordingTracerProvider(t)
			ctx, parent := tp.Tracer("test").Start(context.Background(), "parent")
			_, child := tp.Tracer("test").Start(ctx, "child")
			child.End()
			parent.End()

			spans := recorded()
			require.Len(t, spans, 2)
			assert.Equal(t, "child", spans[0].Name())
			assert.Equal(t, parent.SpanContext().SpanID(), spans[0].Parent().SpanID())
		})

		t.Run("終了していない span は保持しない", func(t *testing.T) {
			t.Parallel()

			tp, recorded := NewRecordingTracerProvider(t)
			_, span := tp.Tracer("test").Start(context.Background(), "open")
			defer span.End()

			assert.Empty(t, recorded())
		})

		t.Run("取り出した span の列は複製で、書き換えても保持側に影響しない", func(t *testing.T) {
			t.Parallel()

			tp, recorded := NewRecordingTracerProvider(t)
			_, span := tp.Tracer("test").Start(context.Background(), "kept")
			span.End()

			first := recorded()
			require.Len(t, first, 1)
			first[0] = nil

			again := recorded()
			require.Len(t, again, 1)
			assert.Equal(t, "kept", again[0].Name())
		})
	})
}

//nolint:paralleltest // otel の global provider を差し替えるため並列化しない
func TestInstallRecordingTracerProvider(t *testing.T) {
	t.Run("正常系", func(t *testing.T) {
		t.Run("global の provider から得た tracer の span を保持する", func(t *testing.T) {
			recorded := InstallRecordingTracerProvider(t)
			_, span := otel.Tracer("test").Start(context.Background(), "global-probe")
			span.End()

			spans := recorded()
			require.Len(t, spans, 1)
			assert.Equal(t, "global-probe", spans[0].Name())
		})

		t.Run("テスト終了時に global の provider を元へ戻す", func(t *testing.T) {
			before := otel.GetTracerProvider()

			t.Run("差し替え中は別の provider になる", func(t *testing.T) {
				InstallRecordingTracerProvider(t)
				assert.NotSame(t, before, otel.GetTracerProvider())
			})

			assert.Same(t, before, otel.GetTracerProvider())
		})
	})
}

func Test_spanRecorder_ExportSpans(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("複数回の export を順に累積する", func(t *testing.T) {
			t.Parallel()

			tp, recorded := NewRecordingTracerProvider(t)
			tracer := tp.Tracer("test")
			_, first := tracer.Start(context.Background(), "first")
			first.End()
			_, second := tracer.Start(context.Background(), "second")
			second.End()
			source := recorded()
			require.Len(t, source, 2)

			rec := &spanRecorder{}
			require.NoError(t, rec.ExportSpans(context.Background(), source[:1]))
			require.NoError(t, rec.ExportSpans(context.Background(), source[1:]))

			require.Len(t, rec.spans, 2)
			assert.Equal(t, "first", rec.spans[0].Name())
			assert.Equal(t, "second", rec.spans[1].Name())
		})
	})
}

func Test_spanRecorder_Shutdown(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("停止は常に成功する", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, (&spanRecorder{}).Shutdown(context.Background()))
		})
	})
}

func TestSpanAttributeValues(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("絞り込みに一致した span の全属性値を返す", func(t *testing.T) {
			t.Parallel()

			tp, recorded := NewRecordingTracerProvider(t)
			tracer := tp.Tracer("test")
			_, hit := tracer.Start(context.Background(), "hit", trace.WithAttributes(attribute.String("url.path", "/a"), attribute.String("k", "v")))
			hit.End()
			_, miss := tracer.Start(context.Background(), "miss", trace.WithAttributes(attribute.String("url.path", "/b"), attribute.String("k", "w")))
			miss.End()

			values := SpanAttributeValues(recorded(), "url.path", "/a")

			assert.ElementsMatch(t, []string{"/a", "v"}, values)
		})

		t.Run("一致する span が無ければ空", func(t *testing.T) {
			t.Parallel()

			tp, recorded := NewRecordingTracerProvider(t)
			_, span := tp.Tracer("test").Start(context.Background(), "other")
			span.End()

			assert.Empty(t, SpanAttributeValues(recorded(), "url.path", "/a"))
		})
	})
}

func Test_hasAttribute(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("keyとvalueの両方が一致するときだけtrue", func(t *testing.T) {
			t.Parallel()

			tp, recorded := NewRecordingTracerProvider(t)
			_, span := tp.Tracer("test").Start(context.Background(), "s", trace.WithAttributes(attribute.String("k", "v")))
			span.End()
			spans := recorded()
			require.Len(t, spans, 1)

			assert.True(t, hasAttribute(spans[0], "k", "v"))
			assert.False(t, hasAttribute(spans[0], "k", "x"))
			assert.False(t, hasAttribute(spans[0], "other", "v"))
		})
	})
}
