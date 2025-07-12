package testUtil

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetTestEnv_SetsENVToTest(t *testing.T) {
	t.Run("ENVに'test'を設定する", func(t *testing.T) {
		t.Setenv("ENV", "")
		SetTestEnv(&testing.M{})

		assert.Equal(t, "test", os.Getenv("ENV"))
	})

	t.Run("引数でnilを渡すとパニックする", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic when passing nil, but did not panic")
			}
		}()

		SetTestEnv(nil)
	})
}
