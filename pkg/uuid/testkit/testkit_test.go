package testkit_test

import (
	"testing"

	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/stretchr/testify/assert"
)

func TestNewTestFromSalt(t *testing.T) {
	t.Parallel()
	salt := "test-salt"
	uuid1 := uuidtestkit.NewTestFromSalt(t, salt)
	uuid2 := uuidtestkit.NewTestFromSalt(t, salt)
	assert.Equal(t, uuid1, uuid2)
	assert.NotEqual(t, uuid1, uuidtestkit.NewTestFromSalt(t, "other-salt"))
}
