package eventlog_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go-boilerplate/internal/infrastructure/dynamodbclient/testkit"
	"go-boilerplate/internal/infrastructure/eventlog"
	"go-boilerplate/internal/observability"
)

func TestNew(t *testing.T) {
	t.Parallel()

	s := eventlog.New(testkit.NewTestClient(t), "realtime_event_log_test", observability.NewNoopTracerFactory(t))
	assert.NotNil(t, s)
}
