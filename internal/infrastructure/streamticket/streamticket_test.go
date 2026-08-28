package streamticket_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go-boilerplate/internal/infrastructure/dynamodbclient/testkit"
	"go-boilerplate/internal/infrastructure/streamticket"
	"go-boilerplate/internal/observability"
)

func TestNew(t *testing.T) {
	t.Parallel()

	s := streamticket.New(testkit.NewTestClient(t), "realtime_stream_ticket_test", observability.NewNoopTracerFactory(t))
	assert.NotNil(t, s)
}
