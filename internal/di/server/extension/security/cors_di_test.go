package security

import (
	"testing"

	"go-boilerplate/internal/config"

	"github.com/stretchr/testify/assert"
)

func TestCORSMiddleware(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	secCfg := config.NewSecurityConfig(cfg)

	out := CORSMiddleware(secCfg)
	assert.Equal(t, corsPriority, out.Middleware.Priority)
	assert.NotNil(t, out.Middleware.Middleware)
	assert.Equal(t, "cors", out.Middleware.Name)
}
