package redaction

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
)

func newSpecWithSchemes(schemes map[string]*openapi3.SecurityScheme) *openapi3.T {
	spec := &openapi3.T{Components: &openapi3.Components{SecuritySchemes: openapi3.SecuritySchemes{}}}
	for name, scheme := range schemes {
		spec.Components.SecuritySchemes[name] = &openapi3.SecuritySchemeRef{Value: scheme}
	}
	return spec
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("渡した名前だけを秘匿する", func(t *testing.T) {
			t.Parallel()
			r := New([]string{"ticket"})
			assert.Equal(t, "/p?ticket="+RedactedValue+"&after=1", r.URI("/p?ticket=secret&after=1"))
		})

		t.Run("名前が空なら何も秘匿しない", func(t *testing.T) {
			t.Parallel()
			r := New(nil)
			assert.Equal(t, "/p?ticket=secret", r.URI("/p?ticket=secret"))
		})
	})
}

func TestFromSpec(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("queryのapiKeyの名前を秘匿する", func(t *testing.T) {
			t.Parallel()
			r := FromSpec(newSpecWithSchemes(map[string]*openapi3.SecurityScheme{
				"StreamTicket": {Type: "apiKey", In: "query", Name: "ticket"},
			}))
			assert.Equal(t, "/p?ticket="+RedactedValue, r.URI("/p?ticket=secret"))
		})
	})
}

func TestSecretQueryParamNames(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("queryのapiKeyだけを名前順で返す", func(t *testing.T) {
			t.Parallel()
			spec := newSpecWithSchemes(map[string]*openapi3.SecurityScheme{
				"Zeta":   {Type: "apiKey", In: "query", Name: "zeta"},
				"Alpha":  {Type: "apiKey", In: "query", Name: "alpha"},
				"Header": {Type: "apiKey", In: "header", Name: "X-Api-Key"},
				"Bearer": {Type: "http", Scheme: "bearer"},
			})
			assert.Equal(t, []string{"alpha", "zeta"}, SecretQueryParamNames(spec))
		})

		t.Run("該当するschemeが無ければ空", func(t *testing.T) {
			t.Parallel()
			spec := newSpecWithSchemes(map[string]*openapi3.SecurityScheme{"Bearer": {Type: "http", Scheme: "bearer"}})
			assert.Empty(t, SecretQueryParamNames(spec))
		})

		t.Run("specがnilなら空", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, SecretQueryParamNames(nil))
		})

		t.Run("componentsが無ければ空", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, SecretQueryParamNames(&openapi3.T{}))
		})

		t.Run("値の無い参照と名前の無いschemeは飛ばす", func(t *testing.T) {
			t.Parallel()
			spec := newSpecWithSchemes(map[string]*openapi3.SecurityScheme{
				"Unnamed": {Type: "apiKey", In: "query"},
			})
			spec.Components.SecuritySchemes["Empty"] = &openapi3.SecuritySchemeRef{}
			spec.Components.SecuritySchemes["Nil"] = nil
			assert.Empty(t, SecretQueryParamNames(spec))
		})
	})
}

func TestRedactor_URI(t *testing.T) {
	t.Parallel()

	r := New([]string{"ticket"})

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("秘匿対象の値だけを置き換え並びを保つ", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "/v1/streams/s?after=3&ticket="+RedactedValue+"&x=y", r.URI("/v1/streams/s?after=3&ticket=abc&x=y"))
		})

		t.Run("同じ名前が複数あればすべて置き換える", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "/p?ticket="+RedactedValue+"&ticket="+RedactedValue, r.URI("/p?ticket=a&ticket=b"))
		})

		t.Run("符号化された名前も復号して判定する", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "/p?%74icket="+RedactedValue, r.URI("/p?%74icket=abc"))
		})

		t.Run("queryが無ければそのまま返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "/p", r.URI("/p"))
		})

		t.Run("値の無い組はそのまま残す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "/p?ticket&a=1", r.URI("/p?ticket&a=1"))
		})

		t.Run("ゼロ値は何も秘匿しない", func(t *testing.T) {
			t.Parallel()
			var zero Redactor
			assert.Equal(t, "/p?ticket=abc", zero.URI("/p?ticket=abc"))
		})

		t.Run("復号できない符号化を含むqueryは全体を置き換える", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "/p?"+RedactedValue, r.URI("/p?%zz=abc&a=1"))
		})

		t.Run("セミコロン区切りのqueryは全体を置き換える", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "/p?"+RedactedValue, r.URI("/p?a=1;ticket=abc"))
		})
	})
}

func TestRedactor_QueryParams(t *testing.T) {
	t.Parallel()

	r := New([]string{"ticket"})

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("秘匿対象の値を件数を保って置き換え元のmapは変えない", func(t *testing.T) {
			t.Parallel()
			in := map[string][]string{"ticket": {"a", "b"}, "after": {"1"}}

			got := r.QueryParams(in)

			assert.Equal(t, map[string][]string{"ticket": {RedactedValue, RedactedValue}, "after": {"1"}}, got)
			assert.Equal(t, map[string][]string{"ticket": {"a", "b"}, "after": {"1"}}, in)
		})

		t.Run("ゼロ値はそのまま返す", func(t *testing.T) {
			t.Parallel()
			var zero Redactor
			in := map[string][]string{"ticket": {"a"}}
			assert.Equal(t, in, zero.QueryParams(in))
		})

		t.Run("nilはnilのまま返す", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, r.QueryParams(nil))
		})
	})
}

func TestRedactor_secret(t *testing.T) {
	t.Parallel()

	r := New([]string{"ticket"})

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("秘匿対象の名前ならtrue", func(t *testing.T) {
			t.Parallel()
			assert.True(t, r.secret("ticket"))
		})

		t.Run("符号化された名前は復号して判定する", func(t *testing.T) {
			t.Parallel()
			assert.True(t, r.secret("%74icket"))
		})

		t.Run("秘匿対象でなければfalse", func(t *testing.T) {
			t.Parallel()
			assert.False(t, r.secret("after"))
		})

		t.Run("復号できない名前は秘匿対象として扱う", func(t *testing.T) {
			t.Parallel()
			assert.True(t, r.secret("%zz"))
		})
	})
}
