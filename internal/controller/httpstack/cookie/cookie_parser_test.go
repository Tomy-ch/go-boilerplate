package cookie

import (
	"testing"

	"go-boilerplate/pkg/ptr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// assertKV は、属性が値あり属性として want を保持することを検証します。
func assertKV(t *testing.T, attrs *cookieAttrs, key, want string) {
	t.Helper()
	v, ok := attrs.kv[key]
	require.True(t, ok)
	require.NotNil(t, v)
	assert.Equal(t, want, *v)
}

// assertFlag は、属性がフラグ属性（値なし）として存在することを検証します。
func assertFlag(t *testing.T, attrs *cookieAttrs, key string) {
	t.Helper()
	v, ok := attrs.kv[key]
	require.True(t, ok)
	assert.Nil(t, v)
}

func Test_parseSetCookie(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("通常のCookieを解析できる", func(t *testing.T) {
			t.Parallel()
			raw := "sessionid=abc123; HttpOnly; Path=/; Domain=example.com; SameSite=Strict; Max-Age=3600"
			name, value, attrs, ok := parseSetCookie(raw)
			require.True(t, ok)
			require.NotNil(t, attrs)
			assert.Equal(t, "sessionid", name)
			assert.Equal(t, "abc123", value)
			assert.Equal(t, []string{"httponly", "path", "domain", "samesite", "max-age"}, attrs.order)
			assertFlag(t, attrs, "httponly")
			assertKV(t, attrs, "path", "/")
			assertKV(t, attrs, "domain", "example.com")
			assertKV(t, attrs, "samesite", "Strict")
			assertKV(t, attrs, "max-age", "3600")
		})

		t.Run("重複属性は最後の値が反映される", func(t *testing.T) {
			t.Parallel()
			raw := "id=1; Path=/first; Path=/second; Secure"
			name, value, attrs, ok := parseSetCookie(raw)
			require.True(t, ok)
			require.NotNil(t, attrs)
			assert.Equal(t, "id", name)
			assert.Equal(t, "1", value)
			assert.Equal(t, []string{"path", "secure"}, attrs.order)
			assertKV(t, attrs, "path", "/second")
			assertFlag(t, attrs, "secure")
		})

		t.Run("空の属性はスキップされる", func(t *testing.T) {
			t.Parallel()
			raw := "id=1;  ; Path=/; HttpOnly"
			name, value, attrs, ok := parseSetCookie(raw)
			require.True(t, ok)
			require.NotNil(t, attrs)
			assert.Equal(t, "id", name)
			assert.Equal(t, "1", value)
			assert.Equal(t, []string{"path", "httponly"}, attrs.order)
			assertKV(t, attrs, "path", "/")
			assertFlag(t, attrs, "httponly")
		})

		t.Run("空値KV属性はフラグと区別され=が保持される", func(t *testing.T) {
			t.Parallel()
			_, _, attrs, ok := parseSetCookie("id=1; Foo=; Bar")
			require.True(t, ok)
			require.NotNil(t, attrs)
			assertKV(t, attrs, "foo", "")
			assertFlag(t, attrs, "bar")
			assert.Equal(t, "id=1; foo=; bar", buildSetCookie("id", "1", attrs))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("空文字は失敗する", func(t *testing.T) {
			t.Parallel()
			_, _, _, ok := parseSetCookie("")
			assert.False(t, ok)
		})

		t.Run("イコール無しは失敗する", func(t *testing.T) {
			t.Parallel()
			_, _, _, ok := parseSetCookie("NoEquals")
			assert.False(t, ok)
		})
	})
}

func Test_splitAttr(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("等号なしはフラグ扱い", func(t *testing.T) {
			t.Parallel()
			k, v, isKV := splitAttr("HttpOnly")
			assert.Equal(t, "HttpOnly", k)
			assert.Empty(t, v)
			assert.False(t, isKV)
		})

		t.Run("等号ありは分割される", func(t *testing.T) {
			t.Parallel()
			k, v, isKV := splitAttr("Key=Value")
			assert.Equal(t, "Key", k)
			assert.Equal(t, "Value", v)
			assert.True(t, isKV)
		})

		t.Run("先頭が等号でも値が取れる", func(t *testing.T) {
			t.Parallel()
			k, v, isKV := splitAttr("=value")
			assert.Empty(t, k)
			assert.Equal(t, "value", v)
			assert.True(t, isKV)
		})
	})
}

func Test_setBoolAttr(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("オンで追加、オフで削除される", func(t *testing.T) {
			t.Parallel()
			attrs := &cookieAttrs{order: []string{}, kv: map[string]*string{}}

			setBoolAttr(attrs, "Secure", true)
			assert.Equal(t, []string{"secure"}, attrs.order)
			assertFlag(t, attrs, "secure")

			// 再度オンにしても重複しない
			setBoolAttr(attrs, "secure", true)
			assert.Len(t, attrs.order, 1)

			// オフにすると削除される
			setBoolAttr(attrs, "SECURE", false)
			_, ok := attrs.kv["secure"]
			assert.False(t, ok)
			assert.Empty(t, attrs.order)
		})
	})
}

func Test_setKVAttr(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("値が設定され更新される", func(t *testing.T) {
			t.Parallel()
			attrs := &cookieAttrs{order: []string{}, kv: map[string]*string{}}

			setKVAttr(attrs, "Path", "/home")
			assert.Equal(t, []string{"path"}, attrs.order)
			assertKV(t, attrs, "path", "/home")

			// 更新
			setKVAttr(attrs, "path", "/home2")
			assert.Len(t, attrs.order, 1)
			assertKV(t, attrs, "path", "/home2")
		})
	})
}

func Test_delAttr(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定キーが削除され順序も保たれる", func(t *testing.T) {
			t.Parallel()
			attrs := &cookieAttrs{order: []string{"a", "b", "c"}, kv: map[string]*string{"a": ptr.To("1"), "b": ptr.To("2"), "c": ptr.To("3")}}

			delAttr(attrs, "b")
			assert.Equal(t, []string{"a", "c"}, attrs.order)
			_, ok := attrs.kv["b"]
			assert.False(t, ok)

			// 存在しないキーの削除は影響がない
			delAttr(attrs, "z")
			assert.Equal(t, []string{"a", "c"}, attrs.order)
		})
	})
}

func Test_buildSetCookie(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("属性の順序と正規化を反映する", func(t *testing.T) {
			t.Parallel()
			attrs := &cookieAttrs{
				order: []string{"httponly", "path", "domain", "somespecial"},
				kv:    map[string]*string{"httponly": nil, "path": ptr.To("/"), "domain": ptr.To("example.com"), "somespecial": ptr.To("v")},
			}
			s := buildSetCookie("id", "1", attrs)
			assert.Equal(t, "id=1; HttpOnly; Path=/; Domain=example.com; somespecial=v", s)
		})

		t.Run("kvに存在しないorderは無視される", func(t *testing.T) {
			t.Parallel()
			attrs := &cookieAttrs{order: []string{"a", "b"}, kv: map[string]*string{"a": ptr.To("va")}}
			s := buildSetCookie("k", "v", attrs)
			assert.Equal(t, "k=v; a=va", s)
		})
	})
}

func Test_canonicalAttrKey(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("httponlyはHttpOnlyに正規化される", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "HttpOnly", canonicalAttrKey("httponly"))
		})

		t.Run("samesiteはSameSiteに正規化される", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "SameSite", canonicalAttrKey("samesite"))
		})

		t.Run("max-ageはMax-Ageに正規化される", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "Max-Age", canonicalAttrKey("max-age"))
		})

		t.Run("secureはSecureに正規化される", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "Secure", canonicalAttrKey("secure"))
		})

		t.Run("domainはDomainに正規化される", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "Domain", canonicalAttrKey("domain"))
		})

		t.Run("pathはPathに正規化される", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "Path", canonicalAttrKey("path"))
		})

		t.Run("expiresはExpiresに正規化される", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "Expires", canonicalAttrKey("expires"))
		})

		t.Run("既知でない属性は元のまま返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "custom", canonicalAttrKey("custom"))
		})
	})
}
