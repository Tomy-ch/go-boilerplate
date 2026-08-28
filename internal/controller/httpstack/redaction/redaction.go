// Package redaction は、ログへ出す前にリクエスト URI と query から資格情報の値を取り除きます。
// 秘匿する名前は OpenAPI の securityScheme（query の apiKey）から導出し、名前を Go 側に持ちません。
package redaction

import (
	"net/url"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// RedactedValue は、秘匿した値の代わりにログへ残す文字列です。
const RedactedValue = "[REDACTED]"

const (
	securitySchemeAPIKey  = "apiKey"
	securitySchemeInQuery = "query"
)

// Redactor は、秘匿対象の query パラメータの値を RedactedValue へ置き換えます。ゼロ値は何も秘匿しません。
type Redactor struct {
	names map[string]struct{}
}

// New は、names を秘匿対象とする Redactor を返します。
func New(names []string) Redactor {
	set := make(map[string]struct{}, len(names))
	for _, n := range names {
		set[n] = struct{}{}
	}
	return Redactor{names: set}
}

// FromSpec は、spec の securityScheme のうち query で受け取る apiKey の名前を秘匿対象とする Redactor を返します。
func FromSpec(spec *openapi3.T) Redactor {
	return New(SecretQueryParamNames(spec))
}

// SecretQueryParamNames は、spec の securityScheme のうち query で受け取る apiKey のパラメータ名を名前順で返します。
func SecretQueryParamNames(spec *openapi3.T) []string {
	if spec == nil || spec.Components == nil {
		return nil
	}

	var names []string
	for _, ref := range spec.Components.SecuritySchemes {
		if ref == nil || ref.Value == nil {
			continue
		}
		if ref.Value.Type == securitySchemeAPIKey && ref.Value.In == securitySchemeInQuery && ref.Value.Name != "" {
			names = append(names, ref.Value.Name)
		}
	}
	sort.Strings(names)

	return names
}

// URI は、raw（path と query を含むリクエスト URI）から秘匿対象パラメータの値を取り除いた文字列を返します。
// query の並びと符号化はそのまま保ちます。標準の構文解析が受け付けない query（`;` 区切りや壊れた符号化）は
// 組ごとに判定できないため、query 全体を置き換えます（fail-closed）。
func (r Redactor) URI(raw string) string {
	if len(r.names) == 0 {
		return raw
	}

	path, query, found := strings.Cut(raw, "?")
	if !found {
		return raw
	}

	if _, err := url.ParseQuery(query); err != nil {
		return path + "?" + RedactedValue
	}

	pairs := strings.Split(query, "&")
	for i, pair := range pairs {
		key, _, hasValue := strings.Cut(pair, "=")
		if !hasValue {
			continue
		}
		if r.secret(key) {
			pairs[i] = key + "=" + RedactedValue
		}
	}

	return path + "?" + strings.Join(pairs, "&")
}

// QueryParams は、秘匿対象パラメータの値を RedactedValue へ置き換えた map を返します。元の map は変更しません。
// 秘匿対象が 1 つも無い場合は params をそのまま返すため、返り値が複製であることは保証しません。
func (r Redactor) QueryParams(params map[string][]string) map[string][]string {
	if len(r.names) == 0 || params == nil {
		return params
	}

	out := make(map[string][]string, len(params))
	for name, values := range params {
		if _, ok := r.names[name]; !ok {
			out[name] = values
			continue
		}
		redacted := make([]string, len(values))
		for i := range values {
			redacted[i] = RedactedValue
		}
		out[name] = redacted
	}

	return out
}

// secret は、符号化されたままの key が秘匿対象かを返します。復号できない key は値も読めないため秘匿対象として扱います。
func (r Redactor) secret(encodedKey string) bool {
	key, err := url.QueryUnescape(encodedKey)
	if err != nil {
		return true
	}
	_, ok := r.names[key]
	return ok
}
