package uuid

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewTestFromSalt(t *testing.T) {
	t.Parallel()
	salt := "test-salt"
	uuid1 := NewTestFromSalt(t, salt)
	uuid2 := NewTestFromSalt(t, salt)
	assert.Equal(t, uuid1, uuid2)
	assert.NotEqual(t, uuid1, NewTestFromSalt(t, "other-salt"))
}
