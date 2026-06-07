package uuid

import (
	"testing"

	guuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromGoogle(t *testing.T) {
	g := guuid.New()
	u := fromGoogle(g)
	assert.Equal(t, g.String(), u.String())
}

func TestToGoogle(t *testing.T) {
	uuid, err := New()
	require.NoError(t, err)

	g := toGoogle(uuid)
	assert.Equal(t, uuid.String(), g.String())
}
