package cookie

import (
	"testing"

	"go-boilerplate/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// assertNoAttr は、属性が存在しないことを検証します。
func assertNoAttr(t *testing.T, attrs *cookieAttrs, key string) {
	t.Helper()
	_, ok := attrs.kv[key]
	assert.False(t, ok)
}

func TestNewSecurityCookie(t *testing.T) {
	t.Parallel()

	t.Run("MockConfigForTest の設定（secure=true/sameSite=Strict/domain=localhost）が反映される", func(t *testing.T) {
		t.Parallel()
		cfg := config.MockConfigForTest(t)
		scfg := config.NewSecureCookieConfig(cfg)
		sec := NewSecurityCookie(scfg)
		require.NotNil(t, sec)

		name, value, attrs, ok := parseSetCookie(sec.RewriteSetCookie("a=1"))
		require.True(t, ok)
		require.NotNil(t, attrs)
		assert.Equal(t, "a", name)
		assert.Equal(t, "1", value) // value 透過
		assertFlag(t, attrs, "httponly")
		assertFlag(t, attrs, "secure")
		assertKV(t, attrs, "samesite", "Strict")
		assertKV(t, attrs, "domain", "localhost")
	})
}

func TestSecurityCookie_RewriteSetCookie(t *testing.T) {
	t.Parallel()

	t.Run("parse失敗で空を返す", func(t *testing.T) {
		t.Parallel()
		sec := &SecurityCookie{applyToAll: true}
		assert.Empty(t, sec.RewriteSetCookie("not-a-cookie"))
	})

	t.Run("name が空（=1）なら空を返す", func(t *testing.T) {
		t.Parallel()
		sec := &SecurityCookie{applyToAll: true}
		assert.Empty(t, sec.RewriteSetCookie("=1"))
	})

	t.Run("applyToAll=false で非対象は元の raw を返す", func(t *testing.T) {
		t.Parallel()
		sec := &SecurityCookie{applyToAll: false, cookieNames: map[string]struct{}{"foo": {}}}
		raw := "bar=1; Domain=example.com"
		assert.Equal(t, raw, sec.RewriteSetCookie(raw))
	})

	t.Run("applyToAll=false で対象名なら書き換える", func(t *testing.T) {
		t.Parallel()
		sec := &SecurityCookie{applyToAll: false, cookieNames: map[string]struct{}{"bar": {}}, forcePath: "/ok"}
		_, _, attrs, ok := parseSetCookie(sec.RewriteSetCookie("bar=1"))
		require.True(t, ok)
		require.NotNil(t, attrs)
		assertKV(t, attrs, "path", "/ok")
	})

	t.Run("skipCookieNames に含まれると元の raw を返す", func(t *testing.T) {
		t.Parallel()
		sec := &SecurityCookie{applyToAll: true, skipCookieNames: map[string]struct{}{"s": {}}}
		raw := "s=1; Path=/"
		assert.Equal(t, raw, sec.RewriteSetCookie(raw))
	})

	t.Run("forceSecure=true で Secure が付与される", func(t *testing.T) {
		t.Parallel()
		v := true
		sec := &SecurityCookie{applyToAll: true, forceSecure: &v}
		_, _, attrs, ok := parseSetCookie(sec.RewriteSetCookie("x=1"))
		require.True(t, ok)
		require.NotNil(t, attrs)
		assertFlag(t, attrs, "secure")
	})

	t.Run("forceSecure=false で既存の Secure が削除される", func(t *testing.T) {
		t.Parallel()
		v := false
		sec := &SecurityCookie{applyToAll: true, forceSecure: &v}
		_, _, attrs, ok := parseSetCookie(sec.RewriteSetCookie("x=1; Secure"))
		require.True(t, ok)
		require.NotNil(t, attrs)
		assertNoAttr(t, attrs, "secure")
	})

	t.Run("forceHTTPOnly が反映される", func(t *testing.T) {
		t.Parallel()
		v := true
		sec := &SecurityCookie{applyToAll: true, forceHTTPOnly: &v}
		_, _, attrs, ok := parseSetCookie(sec.RewriteSetCookie("y=1"))
		require.True(t, ok)
		require.NotNil(t, attrs)
		assertFlag(t, attrs, "httponly")
	})

	t.Run("forceSameSite=None で Secure が付与される", func(t *testing.T) {
		t.Parallel()
		sec := &SecurityCookie{applyToAll: true, forceSameSite: "None", enforceSecureWhenSameSiteNone: true}
		_, _, attrs, ok := parseSetCookie(sec.RewriteSetCookie("z=1"))
		require.True(t, ok)
		require.NotNil(t, attrs)
		assertKV(t, attrs, "samesite", "None")
		assertFlag(t, attrs, "secure")
	})

	t.Run("入力由来の SameSite=None でも Secure が付与される", func(t *testing.T) {
		t.Parallel()
		sec := &SecurityCookie{applyToAll: true, enforceSecureWhenSameSiteNone: true}
		_, _, attrs, ok := parseSetCookie(sec.RewriteSetCookie("z=1; SameSite=None"))
		require.True(t, ok)
		require.NotNil(t, attrs)
		assertFlag(t, attrs, "secure")
	})

	t.Run("enforceSecureWhenSameSiteNone=false なら None でも Secure を付けない", func(t *testing.T) {
		t.Parallel()
		sec := &SecurityCookie{applyToAll: true, forceSameSite: "None", enforceSecureWhenSameSiteNone: false}
		_, _, attrs, ok := parseSetCookie(sec.RewriteSetCookie("z=1"))
		require.True(t, ok)
		require.NotNil(t, attrs)
		assertNoAttr(t, attrs, "secure")
	})

	t.Run("forcePath/forceDomain/forceMaxAge が反映される", func(t *testing.T) {
		t.Parallel()
		ma := 3600
		sec := &SecurityCookie{applyToAll: true, forcePath: "/ok", forceDomain: "d.example", forceMaxAge: &ma}
		_, _, attrs, ok := parseSetCookie(sec.RewriteSetCookie("p=1"))
		require.True(t, ok)
		require.NotNil(t, attrs)
		assertKV(t, attrs, "path", "/ok")
		assertKV(t, attrs, "domain", "d.example")
		assertKV(t, attrs, "max-age", "3600")
	})

	t.Run("__Secure- prefix は Secure を強制する", func(t *testing.T) {
		t.Parallel()
		sec := &SecurityCookie{applyToAll: true}
		_, _, attrs, ok := parseSetCookie(sec.RewriteSetCookie("__Secure-a=1"))
		require.True(t, ok)
		require.NotNil(t, attrs)
		assertFlag(t, attrs, "secure")
	})

	t.Run("__Host- prefix は Secure + Path=/ + Domain 削除 を行う", func(t *testing.T) {
		t.Parallel()
		sec := &SecurityCookie{applyToAll: true}
		_, _, attrs, ok := parseSetCookie(sec.RewriteSetCookie("__Host-cookie=1; Domain=example.com; Path=/foo"))
		require.True(t, ok)
		require.NotNil(t, attrs)
		assertFlag(t, attrs, "secure")
		assertKV(t, attrs, "path", "/")
		assertNoAttr(t, attrs, "domain")
	})
}

func Test_normalizeSameSite(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		want string
	}{
		{"小文字laxはLaxへ正規化", "lax", "Lax"},
		{"大文字混在STRICTはStrictへ正規化", "STRICT", "Strict"},
		{"noneはNoneへ正規化", "none", "None"},
		{"前後空白は除去して正規化", " None ", "None"},
		{"タイポは空を返す", "Strcit", ""},
		{"空文字は空を返す", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, c.want, normalizeSameSite(c.in))
		})
	}
}
