package httpheader

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSensitive(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("資格情報を運ぶヘッダを機微と判定する", func(t *testing.T) {
			t.Parallel()

			for _, name := range []string{"authorization", "proxy-authorization", "cookie", "set-cookie"} {
				assert.Truef(t, IsSensitive(name), "%s を機微と判定していない", name)
			}
		})

		t.Run("資格情報を運ばないヘッダは機微と判定しない", func(t *testing.T) {
			t.Parallel()

			for _, name := range []string{"traceparent", "tracestate", "content-type", "x-request-id"} {
				assert.Falsef(t, IsSensitive(name), "%s を機微と判定している", name)
			}
		})

		t.Run("大小文字の違いを無視して判定する", func(t *testing.T) {
			t.Parallel()

			assert.True(t, IsSensitive("Authorization"))
			assert.True(t, IsSensitive("SET-COOKIE"))
		})

		t.Run("前後の空白を無視して判定する", func(t *testing.T) {
			t.Parallel()

			assert.True(t, IsSensitive(" Authorization"))
			assert.True(t, IsSensitive("cookie\t"))
			assert.True(t, IsSensitive("\n Proxy-Authorization \n"))
		})

		t.Run("空文字は機微と判定しない", func(t *testing.T) {
			t.Parallel()

			assert.False(t, IsSensitive(""))
			assert.False(t, IsSensitive("   "))
		})

		t.Run("機微ヘッダ名を含むだけの別ヘッダは機微と判定しない", func(t *testing.T) {
			t.Parallel()

			assert.False(t, IsSensitive("x-authorization"))
			assert.False(t, IsSensitive("authorization-scheme"))
		})
	})
}
