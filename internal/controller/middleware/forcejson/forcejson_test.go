package forcejson

import (
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func Test_isBlacklistedContentType(t *testing.T) {
	t.Run("何も設定されていない場合はtrueを返す", func(t *testing.T) {
		assert.True(t, isBlacklistedContentType(""))
	})
	t.Run("text/htmlの場合はtrueを返す", func(t *testing.T) {
		assert.True(t, isBlacklistedContentType(echo.MIMETextHTML))
	})
	t.Run("application/jsonの場合はfalseを返す", func(t *testing.T) {
		assert.False(t, isBlacklistedContentType(echo.MIMEApplicationJSON))
	})
	t.Run("application/xmlの場合はfalseを返す", func(t *testing.T) {
		assert.False(t, isBlacklistedContentType(echo.MIMEApplicationXML))
	})
}
