package instancelease_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go-boilerplate/internal/infrastructure/dynamodbclient/testkit"
	"go-boilerplate/internal/infrastructure/instancelease"
	"go-boilerplate/internal/observability"
)

func TestNew(t *testing.T) {
	t.Parallel()

	s := instancelease.New(testkit.NewTestClient(t), "realtime_instance_lease_test", observability.NewNoopTracerFactory(t))
	assert.NotNil(t, s)
}
