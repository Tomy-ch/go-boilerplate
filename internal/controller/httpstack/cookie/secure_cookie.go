// Package cookie は、セキュアなCookie設定を提供します。
package cookie

import (
	"strconv"
	"strings"

	"go-boilerplate/internal/config"
)

// SecurityCookie は Set-Cookie を正規化するための設定です。
type SecurityCookie struct {
	// applyToAll が true の場合は Set-Cookie を全部対象にする
	applyToAll bool

	// cookieNames は、ApplyToAll が false の場合に対象とする Cookie 名の集合です。
	cookieNames map[string]struct{}

	// skipCookieNames は、上書き対象から除外する Cookie 名の集合です。
	skipCookieNames map[string]struct{}

	// forceSecure は、Secure 属性を強制的に付与/削除します（nil は上書きしない）
	forceSecure *bool
	// forceHTTPOnly は、HttpOnly 属性を強制的に付与/削除します（nil は上書きしない）
	forceHTTPOnly *bool

	// forceSameSite は、SameSite 属性を強制的に上書きします。
	// "Lax" / "Strict" / "None" / ""(上書きしない)
	forceSameSite string

	// forcePath は Path 属性を強制的に上書きします。
	// 空文字列の場合は上書きしません。
	forcePath string
	// forceDomain は Domain 属性を強制的に上書きします。
	// 空文字列の場合は上書きしません。
	forceDomain string

	// forceMaxAge は Max-Age 属性を強制的に上書きします（nil は上書きしない）
	forceMaxAge *int

	// enforceSecureWhenSameSiteNone は、SameSite=None の場合に Secure 属性を強制的に付与します。
	enforceSecureWhenSameSiteNone bool
}

// NewSecurityCookie は、設定から SecurityCookie を構築します。
func NewSecurityCookie(
	p *config.SecureCookieConfig,
) *SecurityCookie {
	cfg := &SecurityCookie{
		// デフォルト（boilerplate 推奨値）
		applyToAll: true,

		// HttpOnly は既定で常に付与する（拡張時はここを直接変更する）
		forceHTTPOnly: new(true),

		// SameSite=None のとき Secure を要求（ブラウザ仕様に沿って安全側）
		enforceSecureWhenSameSiteNone: true,

		// 多くのケースで "/" 固定
		forcePath: "/",

		cookieNames:     map[string]struct{}{},
		skipCookieNames: map[string]struct{}{},
		forceMaxAge:     nil,
	}

	cfg.forceSecure = p.Secure()
	// SameSite: 空なら上書きしない、許容値（Lax/Strict/None）へ正規化し非許容値は無視
	if s := normalizeSameSite(p.SameSite()); s != "" {
		cfg.forceSameSite = s
	}
	// Domain: 空なら上書きしない
	// （__Host- を使う場合は Rewrite 内で Domain が落とされる）
	if d := strings.TrimSpace(p.Domain()); d != "" {
		cfg.forceDomain = d
	}

	return cfg
}

// RewriteSetCookie は Set-Cookie ヘッダ値（1本）を cfg に基づいて上書きします。
// 失敗時は空文字を返します（呼び出し側で元の raw を使う想定）。
func (cfg *SecurityCookie) RewriteSetCookie(raw string) string {
	name, value, attrs, ok := parseSetCookie(raw)
	if !ok || name == "" {
		return ""
	}
	if !cfg.targets(name) {
		return raw
	}

	if cfg.forceSecure != nil {
		setBoolAttr(attrs, "secure", *cfg.forceSecure)
	}
	if cfg.forceHTTPOnly != nil {
		setBoolAttr(attrs, "httponly", *cfg.forceHTTPOnly)
	}
	cfg.applySameSite(attrs)
	if cfg.forcePath != "" {
		setKVAttr(attrs, "path", cfg.forcePath)
	}
	if cfg.forceDomain != "" {
		setKVAttr(attrs, "domain", cfg.forceDomain)
	}
	if cfg.forceMaxAge != nil {
		setKVAttr(attrs, "max-age", strconv.Itoa(*cfg.forceMaxAge))
	}
	// expires は Secure 属性との整合性を保つのが困難なため、Max-Age と並行して操作しない。
	cfg.applyNamePrefix(name, attrs)

	return buildSetCookie(name, value, attrs)
}

// targets は、name を書き換え対象とするか判定します。
func (cfg *SecurityCookie) targets(name string) bool {
	if !cfg.applyToAll {
		if _, ok := cfg.cookieNames[name]; !ok {
			return false
		}
	}
	_, skip := cfg.skipCookieNames[name]
	return !skip
}

// applySameSite は SameSite 上書きと、実効 SameSite=None 時の Secure 強制を適用します。
func (cfg *SecurityCookie) applySameSite(attrs *cookieAttrs) {
	if cfg.forceSameSite != "" {
		setKVAttr(attrs, "samesite", cfg.forceSameSite)
	}
	if !cfg.enforceSecureWhenSameSiteNone {
		return
	}
	// 実効 SameSite（強制値優先、無ければ入力値）が None なら Secure を強制する
	effective := cfg.forceSameSite
	if effective == "" {
		if p := attrs.kv["samesite"]; p != nil {
			effective = *p
		}
	}
	if strings.EqualFold(effective, "None") {
		setBoolAttr(attrs, "secure", true)
	}
}

// applyNamePrefix は __Secure-/__Host- prefix の安全要件を適用します。
func (cfg *SecurityCookie) applyNamePrefix(name string, attrs *cookieAttrs) {
	hostPrefix := strings.HasPrefix(name, "__Host-")
	securePrefix := strings.HasPrefix(name, "__Secure-") || hostPrefix
	if securePrefix {
		setBoolAttr(attrs, "secure", true)
	}
	if hostPrefix {
		// __Host- は Secure（securePrefix で付与済み）+ Path=/ + Domain無し
		setKVAttr(attrs, "path", "/")
		delAttr(attrs, "domain")
	}
}

// normalizeSameSite は SameSite 値を許容値（Lax/Strict/None）へ正規化します。非許容値・空は "" を返します。
func normalizeSameSite(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "lax":
		return "Lax"
	case "strict":
		return "Strict"
	case "none":
		return "None"
	default:
		return ""
	}
}
