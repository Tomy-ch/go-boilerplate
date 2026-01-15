package cookie

import (
	"strings"
)

const keyValueAttrSep = "="

// cookieAttrs は Set-Cookie 属性を表現します。
type cookieAttrs struct {
	// 順序をある程度維持するために slice を持つ
	order []string
	kv    map[string]string // key(lower) -> value ("" means flag)
}

// parseSetCookie は Set-Cookie ヘッダ値を解析します。
// 失敗時は ok == false を返します。
func parseSetCookie(raw string) (string, string, *cookieAttrs, bool) {
	parts := strings.Split(raw, ";")
	first := strings.TrimSpace(parts[0])
	eq := strings.Index(first, keyValueAttrSep)
	if eq <= 0 {
		return "", "", nil, false
	}

	name := strings.TrimSpace(first[:eq])
	value := strings.TrimSpace(first[eq+1:]) // value はそのまま（quoted/encoded を壊さない）

	attrs := &cookieAttrs{order: make([]string, 0, len(parts)-1), kv: make(map[string]string, len(parts)-1)}
	for _, p := range parts[1:] {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		k, v, isKV := splitAttr(p)
		kl := strings.ToLower(k)
		if _, exists := attrs.kv[kl]; !exists {
			attrs.order = append(attrs.order, kl)
		}
		if isKV {
			attrs.kv[kl] = v
		} else {
			attrs.kv[kl] = ""
		}
	}
	return name, value, attrs, true
}

// splitAttr は 属性文字列を key/value に分割します。
func splitAttr(s string) (string, string, bool) {
	eq := strings.Index(s, keyValueAttrSep)
	if eq < 0 {
		return strings.TrimSpace(s), "", false
	}
	return strings.TrimSpace(s[:eq]), strings.TrimSpace(s[eq+1:]), true
}

// setBoolAttr は ブール属性を設定/削除します。
func setBoolAttr(attrs *cookieAttrs, key string, on bool) {
	key = strings.ToLower(key)
	if on {
		if _, exists := attrs.kv[key]; !exists {
			attrs.order = append(attrs.order, key)
		}
		attrs.kv[key] = ""
		return
	}
	// off
	delAttr(attrs, key)
}

// setKVAttr は key/value 属性を設定します。
func setKVAttr(attrs *cookieAttrs, key, val string) {
	key = strings.ToLower(key)
	if _, exists := attrs.kv[key]; !exists {
		attrs.order = append(attrs.order, key)
	}
	attrs.kv[key] = val
}

// delAttr は 属性を削除します。
func delAttr(attrs *cookieAttrs, key string) {
	key = strings.ToLower(key)
	if _, exists := attrs.kv[key]; !exists {
		return
	}
	delete(attrs.kv, key)
	// order からも消す
	out := attrs.order[:0]
	for _, k := range attrs.order {
		if k != key {
			out = append(out, k)
		}
	}
	attrs.order = out
}

// buildSetCookie は Set-Cookie ヘッダ値を構築します。
func buildSetCookie(name, value string, attrs *cookieAttrs) string {
	var b strings.Builder
	b.WriteString(name)
	b.WriteString(keyValueAttrSep)
	b.WriteString(value)

	// 既存の順序を尊重しつつ出力
	for _, k := range attrs.order {
		v, ok := attrs.kv[k]
		if !ok {
			continue
		}
		b.WriteString("; ")
		b.WriteString(canonicalAttrKey(k))
		if v != "" {
			b.WriteString(keyValueAttrSep)
			b.WriteString(v)
		}
	}
	return b.String()
}

// canonicalAttrKey は 属性キーの正規化を行います。
func canonicalAttrKey(k string) string {
	switch strings.ToLower(k) {
	case "httponly":
		return "HttpOnly"
	case "samesite":
		return "SameSite"
	case "max-age":
		return "Max-Age"
	case "secure":
		return "Secure"
	case "domain":
		return "Domain"
	case "path":
		return "Path"
	case "expires":
		return "Expires"
	default:
		// それ以外はそのまま（先頭だけ大文字等は揃えない）
		return k
	}
}
