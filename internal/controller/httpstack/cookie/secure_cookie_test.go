package cookie

import (
	"testing"

	"go-boilerplate/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSecurityCookie(t *testing.T) {
	t.Parallel()

	t.Run("MockConfigForTest の設定が反映される", func(t *testing.T) {
		t.Parallel()
		cfg := config.MockConfigForTest(t)
		scfg := config.NewSecureCookieConfig(cfg)
		sec := NewSecurityCookie(scfg)
		require.NotNil(t, sec)

		// MockConfigForTest では secure=true, sameSite=Strict, domain=localhost が設定されている
		out := sec.RewriteSetCookie("a=1")
		// 解析して属性を確認する
		name, _, attrs, ok := parseSetCookie(out)
		assert.True(t, ok)
		assert.Equal(t, "a", name)
		// hasHTTPOnly はデフォルトで有効 (constructorで ptr.To(true) が入るため)
		_, hasHTTPOnly := attrs.kv["httponly"]
		assert.True(t, hasHTTPOnly)
		// SameSite は Strict になる
		v, hasSameSite := attrs.kv["samesite"]
		assert.True(t, hasSameSite)
		assert.Equal(t, "Strict", v)
		// Domain は MockConfigForTest の値が反映される
		dv, hasDomain := attrs.kv["domain"]
		assert.True(t, hasDomain)
		assert.Equal(t, "localhost", dv)
	})
}

func Test_SecurityCookie_RewriteSetCookie(t *testing.T) {
	t.Parallel()

	t.Run("parse失敗で空を返す", func(t *testing.T) {
		t.Parallel()
		sec := &SecurityCookie{applyToAll: true}
		res := sec.RewriteSetCookie("not-a-cookie")
		require.Empty(t, res)
	})

	t.Run("applyToAll=false で非対象は元の raw を返す", func(t *testing.T) {
		t.Parallel()
		sec := &SecurityCookie{applyToAll: false, cookieNames: map[string]struct{}{"foo": {}}}
		raw := "bar=1; Domain=example.com"
		res := sec.RewriteSetCookie(raw)
		assert.Equal(t, raw, res)
	})

	t.Run("skipCookieNames に含まれると元の raw を返す", func(t *testing.T) {
		t.Parallel()
		sec := &SecurityCookie{applyToAll: true, skipCookieNames: map[string]struct{}{"s": {}}}
		raw := "s=1; Path=/"
		res := sec.RewriteSetCookie(raw)
		assert.Equal(t, raw, res)
	})

	t.Run("forceSecure の true/false が反映される", func(t *testing.T) {
		t.Parallel()
		// true の時
		t.Run("true", func(t *testing.T) {
			t.Parallel()
			v := true
			sec := &SecurityCookie{applyToAll: true, forceSecure: &v}
			out := sec.RewriteSetCookie("x=1")
			_, _, attrs, ok := parseSetCookie(out)
			assert.True(t, ok)
			_, has := attrs.kv["secure"]
			assert.True(t, has)
		})
		// false の時
		t.Run("false", func(t *testing.T) {
			t.Parallel()
			v := false
			// input に secure が元々ついていた場合、false にすると削除される
			sec := &SecurityCookie{applyToAll: true, forceSecure: &v}
			out := sec.RewriteSetCookie("x=1; Secure")
			_, _, attrs, ok := parseSetCookie(out)
			assert.True(t, ok)
			_, has := attrs.kv["secure"]
			assert.False(t, has)
		})
	})

	t.Run("forceHTTPOnly が反映される", func(t *testing.T) {
		t.Parallel()
		v := true
		sec := &SecurityCookie{applyToAll: true, forceHTTPOnly: &v}
		out := sec.RewriteSetCookie("y=1")
		_, _, attrs, ok := parseSetCookie(out)
		assert.True(t, ok)
		_, has := attrs.kv["httponly"]
		assert.True(t, has)
	})

	t.Run("forceSameSite=None -> Secure が付与される (enforceSecureWhenSameSiteNone)", func(t *testing.T) {
		t.Parallel()
		sec := &SecurityCookie{applyToAll: true, forceSameSite: "None", enforceSecureWhenSameSiteNone: true}
		out := sec.RewriteSetCookie("z=1")
		_, _, attrs, ok := parseSetCookie(out)
		assert.True(t, ok)
		v, has := attrs.kv["samesite"]
		assert.True(t, has)
		assert.Equal(t, "None", v)
		_, hasSecure := attrs.kv["secure"]
		assert.True(t, hasSecure)
	})

	t.Run("forcePath/forceDomain/forceMaxAge が反映される", func(t *testing.T) {
		t.Parallel()
		ma := 3600
		sec := &SecurityCookie{applyToAll: true, forcePath: "/ok", forceDomain: "d.example", forceMaxAge: &ma}
		out := sec.RewriteSetCookie("p=1")
		_, _, attrs, ok := parseSetCookie(out)
		assert.True(t, ok)
		pv, ph := attrs.kv["path"]
		assert.True(t, ph)
		assert.Equal(t, "/ok", pv)
		dv, dh := attrs.kv["domain"]
		assert.True(t, dh)
		assert.Equal(t, "d.example", dv)
		mv, mh := attrs.kv["max-age"]
		assert.True(t, mh)
		assert.Equal(t, "3600", mv)
	})

	t.Run("__Secure- prefix は Secure を強制する", func(t *testing.T) {
		t.Parallel()
		sec := &SecurityCookie{applyToAll: true}
		out := sec.RewriteSetCookie("__Secure-a=1")
		_, _, attrs, ok := parseSetCookie(out)
		assert.True(t, ok)
		_, has := attrs.kv["secure"]
		assert.True(t, has)
	})

	t.Run("__Host- prefix は Secure + Path=/ + Domain 削除 を行う", func(t *testing.T) {
		t.Parallel()
		sec := &SecurityCookie{applyToAll: true}
		raw := "__Host-cookie=1; Domain=example.com; Path=/foo"
		out := sec.RewriteSetCookie(raw)
		_, _, attrs, ok := parseSetCookie(out)
		assert.True(t, ok)
		// secure
		_, hasSecure := attrs.kv["secure"]
		assert.True(t, hasSecure)
		// path は / に置き換わる
		pv, hasPath := attrs.kv["path"]
		assert.True(t, hasPath)
		assert.Equal(t, "/", pv)
		// domain は削除されている
		_, hasDomain := attrs.kv["domain"]
		assert.False(t, hasDomain)
	})
}
