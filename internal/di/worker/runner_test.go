package worker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_validateShutdownGrace(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("drainがgrace未満なら成功する", func(t *testing.T) {
			t.Parallel()

			require.NoError(t, validateShutdownGrace(30*time.Second, 45*time.Second))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("drainがgraceより大きいとErrInvalidShutdownGraceを返す", func(t *testing.T) {
			t.Parallel()

			err := validateShutdownGrace(60*time.Second, 45*time.Second)
			require.ErrorIs(t, err, ErrInvalidShutdownGrace)
			assert.ErrorContains(t, err, "WORKER_DRAIN_TIMEOUT")
		})

		t.Run("drainとgraceが等しいとErrInvalidShutdownGraceを返す", func(t *testing.T) {
			t.Parallel()

			err := validateShutdownGrace(45*time.Second, 45*time.Second)
			require.ErrorIs(t, err, ErrInvalidShutdownGrace)
		})
	})
}
