package binder

import (
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系_Echoに既定Binderが設定される", func(t *testing.T) {
		t.Parallel()
		e := echo.New()
		New(e)
		assert.IsType(t, &echo.DefaultBinder{}, e.Binder)
	})
}

func TestNewBinder(t *testing.T) {
	t.Parallel()

	t.Run("正常系_既定Binderを生成する", func(t *testing.T) {
		t.Parallel()
		assert.IsType(t, &echo.DefaultBinder{}, NewBinder())
	})
}
