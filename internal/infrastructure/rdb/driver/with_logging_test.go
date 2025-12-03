package driver

import (
	"context"
	"testing"
	"time"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/logging"
	"boilerplate-go/internal/observability"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestBuildSQLLogFields(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	obsCfg := config.NewObservabilityConfig(cfg)
	lf := logging.NewLogFields(obsCfg)

	ctx := context.Background()
	funcName := "TestBuildSQLLogFields"
	query := "SELECT * FROM users"
	expectedDuration := time.Duration(100 * time.Millisecond)
	expectedLatency := float64(expectedDuration) / float64(time.Millisecond)

	expected := []zap.Field{
		zap.String(logging.LayerKey, layer),
		zap.String(logging.PackageKey, pkg),
		zap.String(logging.FunctionKey, funcName),
		zap.String(logging.SpanNameKey, observability.BuildSpanName(layer, pkg, funcName)),
		zap.String(logging.RawQueryKey, query),
		zap.String(logging.QueryCompactKey, query),
		zap.Float64(logging.LatencyKey, expectedLatency),
	}

	dwl := &dbWithLogging{
		provider: &loggingDBProvider{
			lf: lf,
		},
	}

	actual := dwl.buildSQLLogFields(ctx, funcName, query, expectedDuration, nil, nil)
	require.Equal(t, expected, actual)
}
