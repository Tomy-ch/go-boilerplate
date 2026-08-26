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

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("MockConfigForTestの設定（secure=true/sameSite=Strict/domain=localhost）が反映される", func(t *testing.T) {
			t.Parallel()
			cfg := config.MockConfigForTest(t)
			scfg := config.NewSecureCookieConfig(cfg)
			sec := NewSecurityCookie(scfg)
			require.NotNil(t, sec)

			name, value, attrs, ok := parseSetCookie(sec.RewriteSetCookie("a=1"))
			require.True(t, ok)
			require.NotNil(t, attrs)
			assert.Equal(t, "a", name)
			assert.Equal(t, "1", value)
			assertFlag(t, attrs, "httponly")
			assertFlag(t, attrs, "secure")
			assertKV(t, attrs, "samesite", "Strict")
			assertKV(t, attrs, "domain", "localhost")
		})

		t.Run("SameSiteが非許容値の場合はSameSite上書きを行わない", func(t *testing.T) {
			t.Parallel()
			cfg := config.MockConfigForTest(t)
			scfg := config.NewSecureCookieConfig(cfg)
			scfg.SetSameSite(t, "bogus")
			sec := NewSecurityCookie(scfg)

			_, _, attrs, ok := parseSetCookie(sec.RewriteSetCookie("a=1"))
			require.True(t, ok)
			require.NotNil(t, attrs)
			assertNoAttr(t, attrs, "samesite")
		})

		t.Run("Domainが空の場合はDomain上書きを行わない", func(t *testing.T) {
			t.Parallel()
			cfg := config.MockConfigForTest(t)
			scfg := config.NewSecureCookieConfig(cfg)
			scfg.SetDomain(t, "  ")
			sec := NewSecurityCookie(scfg)

			_, _, attrs, ok := parseSetCookie(sec.RewriteSetCookie("a=1"))
			require.True(t, ok)
			require.NotNil(t, attrs)
			assertNoAttr(t, attrs, "domain")
		})
	})
}

func TestSecurityCookie_RewriteSetCookie(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("applyToAll=falseで非対象は元のrawを返す", func(t *testing.T) {
			t.Parallel()
			sec := &SecurityCookie{applyToAll: false, cookieNames: map[string]struct{}{"foo": {}}}
			raw := "bar=1; Domain=example.com"
			assert.Equal(t, raw, sec.RewriteSetCookie(raw))
		})

		t.Run("applyToAll=falseで対象名なら書き換える", func(t *testing.T) {
			t.Parallel()
			sec := &SecurityCookie{applyToAll: false, cookieNames: map[string]struct{}{"bar": {}}, forcePath: "/ok"}
			_, _, attrs, ok := parseSetCookie(sec.RewriteSetCookie("bar=1"))
			require.True(t, ok)
			require.NotNil(t, attrs)
			assertKV(t, attrs, "path", "/ok")
		})

		t.Run("skipCookieNamesに含まれると元のrawを返す", func(t *testing.T) {
			t.Parallel()
			sec := &SecurityCookie{applyToAll: true, skipCookieNames: map[string]struct{}{"s": {}}}
			raw := "s=1; Path=/"
			assert.Equal(t, raw, sec.RewriteSetCookie(raw))
		})

		t.Run("forceSecure=trueでSecureが付与される", func(t *testing.T) {
			t.Parallel()
			v := true
			sec := &SecurityCookie{applyToAll: true, forceSecure: &v}
			_, _, attrs, ok := parseSetCookie(sec.RewriteSetCookie("x=1"))
			require.True(t, ok)
			require.NotNil(t, attrs)
			assertFlag(t, attrs, "secure")
		})

		t.Run("forceSecure=falseで既存のSecureが削除される", func(t *testing.T) {
			t.Parallel()
			v := false
			sec := &SecurityCookie{applyToAll: true, forceSecure: &v}
			_, _, attrs, ok := parseSetCookie(sec.RewriteSetCookie("x=1; Secure"))
			require.True(t, ok)
			require.NotNil(t, attrs)
			assertNoAttr(t, attrs, "secure")
		})

		t.Run("forceHTTPOnlyが反映される", func(t *testing.T) {
			t.Parallel()
			v := true
			sec := &SecurityCookie{applyToAll: true, forceHTTPOnly: &v}
			_, _, attrs, ok := parseSetCookie(sec.RewriteSetCookie("y=1"))
			require.True(t, ok)
			require.NotNil(t, attrs)
			assertFlag(t, attrs, "httponly")
		})

		t.Run("forceHTTPOnly=falseで既存のHttpOnlyが削除される", func(t *testing.T) {
			t.Parallel()
			v := false
			sec := &SecurityCookie{applyToAll: true, forceHTTPOnly: &v}
			_, _, attrs, ok := parseSetCookie(sec.RewriteSetCookie("y=1; HttpOnly"))
			require.True(t, ok)
			require.NotNil(t, attrs)
			assertNoAttr(t, attrs, "httponly")
		})

		t.Run("forceSameSite=NoneでSecureが付与される", func(t *testing.T) {
			t.Parallel()
			sec := &SecurityCookie{applyToAll: true, forceSameSite: "None", enforceSecureWhenSameSiteNone: true}
			_, _, attrs, ok := parseSetCookie(sec.RewriteSetCookie("z=1"))
			require.True(t, ok)
			require.NotNil(t, attrs)
			assertKV(t, attrs, "samesite", "None")
			assertFlag(t, attrs, "secure")
		})

		t.Run("入力由来のSameSite=NoneでもSecureが付与される", func(t *testing.T) {
			t.Parallel()
			sec := &SecurityCookie{applyToAll: true, enforceSecureWhenSameSiteNone: true}
			_, _, attrs, ok := parseSetCookie(sec.RewriteSetCookie("z=1; SameSite=None"))
			require.True(t, ok)
			require.NotNil(t, attrs)
			assertFlag(t, attrs, "secure")
		})

		t.Run("enforceSecureWhenSameSiteNone=falseならNoneでもSecureを付けない", func(t *testing.T) {
			t.Parallel()
			sec := &SecurityCookie{applyToAll: true, forceSameSite: "None", enforceSecureWhenSameSiteNone: false}
			_, _, attrs, ok := parseSetCookie(sec.RewriteSetCookie("z=1"))
			require.True(t, ok)
			require.NotNil(t, attrs)
			assertNoAttr(t, attrs, "secure")
		})

		t.Run("forcePath/forceDomain/forceMaxAgeが反映される", func(t *testing.T) {
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

		t.Run("__Secure-prefixはSecureを強制する", func(t *testing.T) {
			t.Parallel()
			sec := &SecurityCookie{applyToAll: true}
			_, _, attrs, ok := parseSetCookie(sec.RewriteSetCookie("__Secure-a=1"))
			require.True(t, ok)
			require.NotNil(t, attrs)
			assertFlag(t, attrs, "secure")
		})

		t.Run("__Host-prefixはSecure+Path=/+Domain削除を行う", func(t *testing.T) {
			t.Parallel()
			sec := &SecurityCookie{applyToAll: true}
			_, _, attrs, ok := parseSetCookie(sec.RewriteSetCookie("__Host-cookie=1; Domain=example.com; Path=/foo"))
			require.True(t, ok)
			require.NotNil(t, attrs)
			assertFlag(t, attrs, "secure")
			assertKV(t, attrs, "path", "/")
			assertNoAttr(t, attrs, "domain")
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("parse失敗で空を返す", func(t *testing.T) {
			t.Parallel()
			sec := &SecurityCookie{applyToAll: true}
			assert.Empty(t, sec.RewriteSetCookie("not-a-cookie"))
		})

		t.Run("nameが空（=1）なら空を返す", func(t *testing.T) {
			t.Parallel()
			sec := &SecurityCookie{applyToAll: true}
			assert.Empty(t, sec.RewriteSetCookie("=1"))
		})
	})
}

func TestSecurityCookie_targets(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("applyToAll=trueかつskip外は対象になる", func(t *testing.T) {
			t.Parallel()
			cfg := &SecurityCookie{applyToAll: true}
			assert.True(t, cfg.targets("anything"))
		})

		t.Run("applyToAll=falseでcookieNamesに含まれれば対象になる", func(t *testing.T) {
			t.Parallel()
			cfg := &SecurityCookie{applyToAll: false, cookieNames: map[string]struct{}{"foo": {}}}
			assert.True(t, cfg.targets("foo"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("applyToAll=falseでcookieNamesに含まれなければ対象外", func(t *testing.T) {
			t.Parallel()
			cfg := &SecurityCookie{applyToAll: false, cookieNames: map[string]struct{}{"foo": {}}}
			assert.False(t, cfg.targets("bar"))
		})

		t.Run("skipCookieNamesに含まれれば対象外", func(t *testing.T) {
			t.Parallel()
			cfg := &SecurityCookie{applyToAll: true, skipCookieNames: map[string]struct{}{"skip": {}}}
			assert.False(t, cfg.targets("skip"))
		})
	})
}

func TestSecurityCookie_applySameSite(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("forceSameSiteが設定されていれば上書きする", func(t *testing.T) {
			t.Parallel()
			cfg := &SecurityCookie{forceSameSite: "Lax"}
			attrs := &cookieAttrs{order: []string{}, kv: map[string]*string{}}

			cfg.applySameSite(attrs)
			assertKV(t, attrs, "samesite", "Lax")
			assertNoAttr(t, attrs, "secure")
		})

		t.Run("実効SameSite=NoneかつenforceでSecureを強制する", func(t *testing.T) {
			t.Parallel()
			cfg := &SecurityCookie{forceSameSite: "None", enforceSecureWhenSameSiteNone: true}
			attrs := &cookieAttrs{order: []string{}, kv: map[string]*string{}}

			cfg.applySameSite(attrs)
			assertKV(t, attrs, "samesite", "None")
			assertFlag(t, attrs, "secure")
		})

		t.Run("入力由来のSameSite=NoneでもSecureを強制する", func(t *testing.T) {
			t.Parallel()
			cfg := &SecurityCookie{enforceSecureWhenSameSiteNone: true}
			attrs := &cookieAttrs{order: []string{"samesite"}, kv: map[string]*string{"samesite": new("None")}}

			cfg.applySameSite(attrs)
			assertFlag(t, attrs, "secure")
		})

		t.Run("enforceが無効ならNoneでもSecureを付けない", func(t *testing.T) {
			t.Parallel()
			cfg := &SecurityCookie{forceSameSite: "None", enforceSecureWhenSameSiteNone: false}
			attrs := &cookieAttrs{order: []string{}, kv: map[string]*string{}}

			cfg.applySameSite(attrs)
			assertNoAttr(t, attrs, "secure")
		})
	})
}

func TestSecurityCookie_applyNamePrefix(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("prefix無しは何も変更しない", func(t *testing.T) {
			t.Parallel()
			cfg := &SecurityCookie{}
			attrs := &cookieAttrs{order: []string{"domain"}, kv: map[string]*string{"domain": new("example.com")}}

			cfg.applyNamePrefix("plain", attrs)
			assertNoAttr(t, attrs, "secure")
			assertKV(t, attrs, "domain", "example.com")
		})

		t.Run("__Secure-prefixはSecureを強制する", func(t *testing.T) {
			t.Parallel()
			cfg := &SecurityCookie{}
			attrs := &cookieAttrs{order: []string{}, kv: map[string]*string{}}

			cfg.applyNamePrefix("__Secure-a", attrs)
			assertFlag(t, attrs, "secure")
		})

		t.Run("__Host-prefixはSecure付与とPath=/上書きとDomain削除を行う", func(t *testing.T) {
			t.Parallel()
			cfg := &SecurityCookie{}
			attrs := &cookieAttrs{order: []string{"domain", "path"}, kv: map[string]*string{"domain": new("example.com"), "path": new("/foo")}}

			cfg.applyNamePrefix("__Host-a", attrs)
			assertFlag(t, attrs, "secure")
			assertKV(t, attrs, "path", "/")
			assertNoAttr(t, attrs, "domain")
		})
	})
}

func Test_normalizeSameSite(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("小文字laxはLaxへ正規化される", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "Lax", normalizeSameSite("lax"))
		})

		t.Run("大文字混在STRICTはStrictへ正規化される", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "Strict", normalizeSameSite("STRICT"))
		})

		t.Run("noneはNoneへ正規化される", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "None", normalizeSameSite("none"))
		})

		t.Run("前後空白は除去して正規化される", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "None", normalizeSameSite(" None "))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("タイポは空を返す", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, normalizeSameSite("Strcit"))
		})

		t.Run("空文字は空を返す", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, normalizeSameSite(""))
		})
	})
}
