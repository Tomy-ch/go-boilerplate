package cookie

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_parseSetCookie(t *testing.T) {
	t.Parallel()

	t.Run("空文字は失敗する", func(t *testing.T) {
		t.Parallel()
		_, _, _, ok := parseSetCookie("")
		require.False(t, ok)
	})

	t.Run("イコール無しは失敗する", func(t *testing.T) {
		t.Parallel()
		_, _, _, ok := parseSetCookie("NoEquals")
		require.False(t, ok)
	})

	t.Run("通常のCookieを解析できる", func(t *testing.T) {
		t.Parallel()
		raw := "sessionid=abc123; HttpOnly; Path=/; Domain=example.com; SameSite=Strict; Max-Age=3600"
		name, value, attrs, ok := parseSetCookie(raw)
		require.True(t, ok)
		require.Equal(t, "sessionid", name)
		require.Equal(t, "abc123", value)
		require.NotNil(t, attrs)
		require.Equal(t, []string{"httponly", "path", "domain", "samesite", "max-age"}, attrs.order)
		require.Empty(t, attrs.kv["httponly"])
		require.Equal(t, "/", attrs.kv["path"])
		require.Equal(t, "example.com", attrs.kv["domain"])
		require.Equal(t, "Strict", attrs.kv["samesite"])
		require.Equal(t, "3600", attrs.kv["max-age"])
	})

	t.Run("重複属性は最後の値が反映される", func(t *testing.T) {
		t.Parallel()
		raw := "id=1; Path=/first; Path=/second; Secure"
		name, value, attrs, ok := parseSetCookie(raw)
		require.True(t, ok)
		require.Equal(t, "id", name)
		require.Equal(t, "1", value)
		require.Equal(t, []string{"path", "secure"}, attrs.order)
		require.Equal(t, "/second", attrs.kv["path"])
		require.Empty(t, attrs.kv["secure"])
	})

	t.Run("空の属性はスキップされる", func(t *testing.T) {
		t.Parallel()
		raw := "id=1;  ; Path=/; HttpOnly"
		name, value, attrs, ok := parseSetCookie(raw)
		require.True(t, ok)
		require.Equal(t, "id", name)
		require.Equal(t, "1", value)
		require.Equal(t, []string{"path", "httponly"}, attrs.order)
		require.Equal(t, "/", attrs.kv["path"])
		require.Empty(t, attrs.kv["httponly"])
	})
}

func Test_splitAttr(t *testing.T) {
	t.Parallel()

	t.Run("等号なしはフラグ扱い", func(t *testing.T) {
		t.Parallel()
		k, v, isKV := splitAttr("HttpOnly")
		require.Equal(t, "HttpOnly", k)
		require.Empty(t, v)
		require.False(t, isKV)
	})

	t.Run("等号ありは分割される", func(t *testing.T) {
		t.Parallel()
		k, v, isKV := splitAttr("Key=Value")
		require.Equal(t, "Key", k)
		require.Equal(t, "Value", v)
		require.True(t, isKV)
	})

	t.Run("先頭が等号でも値が取れる", func(t *testing.T) {
		t.Parallel()
		k, v, isKV := splitAttr("=value")
		require.Empty(t, k)
		require.Equal(t, "value", v)
		require.True(t, isKV)
	})
}

func Test_setBoolAttr(t *testing.T) {
	t.Parallel()

	t.Run("オンで追加、オフで削除される", func(t *testing.T) {
		t.Parallel()
		attrs := &cookieAttrs{order: []string{}, kv: map[string]string{}}

		setBoolAttr(attrs, "Secure", true)
		require.Equal(t, []string{"secure"}, attrs.order)
		v, ok := attrs.kv["secure"]
		require.True(t, ok)
		require.Empty(t, v)

		// 再度オンにしても重複しない
		setBoolAttr(attrs, "secure", true)
		require.Len(t, attrs.order, 1)

		// オフにすると削除される
		setBoolAttr(attrs, "SECURE", false)
		_, ok = attrs.kv["secure"]
		require.False(t, ok)
		require.Empty(t, attrs.order)
	})
}

func Test_setKVAttr(t *testing.T) {
	t.Parallel()

	t.Run("値が設定され更新される", func(t *testing.T) {
		t.Parallel()
		attrs := &cookieAttrs{order: []string{}, kv: map[string]string{}}

		setKVAttr(attrs, "Path", "/home")
		require.Equal(t, []string{"path"}, attrs.order)
		require.Equal(t, "/home", attrs.kv["path"])

		// 更新
		setKVAttr(attrs, "path", "/home2")
		require.Len(t, attrs.order, 1)
		require.Equal(t, "/home2", attrs.kv["path"])
	})
}

func Test_delAttr(t *testing.T) {
	t.Parallel()

	t.Run("指定キーが削除され順序も保たれる", func(t *testing.T) {
		t.Parallel()
		attrs := &cookieAttrs{order: []string{"a", "b", "c"}, kv: map[string]string{"a": "1", "b": "2", "c": "3"}}

		delAttr(attrs, "b")
		require.Equal(t, []string{"a", "c"}, attrs.order)
		_, ok := attrs.kv["b"]
		require.False(t, ok)

		// 存在しないキーの削除は影響がない
		delAttr(attrs, "z")
		require.Equal(t, []string{"a", "c"}, attrs.order)
	})
}

func Test_buildSetCookie(t *testing.T) {
	t.Parallel()

	t.Run("属性の順序と正規化を反映する", func(t *testing.T) {
		t.Parallel()
		attrs := &cookieAttrs{
			order: []string{"httponly", "path", "domain", "somespecial"},
			kv:    map[string]string{"httponly": "", "path": "/", "domain": "example.com", "somespecial": "v"},
		}
		s := buildSetCookie("id", "1", attrs)
		require.Equal(t, "id=1; HttpOnly; Path=/; Domain=example.com; somespecial=v", s)
	})

	t.Run("kvに存在しないorderは無視される", func(t *testing.T) {
		t.Parallel()
		attrs := &cookieAttrs{order: []string{"a", "b"}, kv: map[string]string{"a": "va"}}
		s := buildSetCookie("k", "v", attrs)
		require.Equal(t, "k=v; a=va", s)
	})
}

func Test_canonicalAttrKey(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   string
		want string
	}{
		{"httponly", "HttpOnly"},
		{"HttpOnly", "HttpOnly"},
		{"samesite", "SameSite"},
		{"max-age", "Max-Age"},
		{"secure", "Secure"},
		{"domain", "Domain"},
		{"path", "Path"},
		{"expires", "Expires"},
		{"custom", "custom"},
	}

	for _, c := range cases {
		c := c
		t.Run(c.in, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, c.want, canonicalAttrKey(c.in))
		})
	}
}
