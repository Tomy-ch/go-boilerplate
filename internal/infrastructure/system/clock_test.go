package system

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewClock(t *testing.T) {
	t.Parallel()

	c := NewClock()
	require.NotNil(t, c)
}

func TestClockNow(t *testing.T) {
	t.Parallel()
	now := NewClock().Now()
	require.WithinDuration(t, time.Now(), now, time.Second)
}
