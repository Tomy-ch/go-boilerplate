package driver

import (
	"context"
	"testing"
	"time"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/logging"

	"github.com/stretchr/testify/require"
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

	dwl := &dbWithLogging{
		provider: &loggingDBProvider{
			lf: lf,
		},
	}

	actual := dwl.buildSQLLogFields(ctx, funcName, query, expectedDuration, nil, nil)
	require.NotEmpty(t, actual)
}
