package cookie

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"strings"
)

const headerSetCookie = "Set-Cookie"

type cookieRewriteWriter struct {
	orig http.ResponseWriter
	cfg  *SecurityCookie

	hdr      http.Header
	wroteHdr bool
}

// newCookieRewriteWriter は cookieRewriteWriter を構築します。
func newCookieRewriteWriter(orig http.ResponseWriter, cfg *SecurityCookie) *cookieRewriteWriter {
	return &cookieRewriteWriter{
		orig: orig,
		cfg:  cfg,
		hdr:  make(http.Header),
	}
}

// Header はヘッダを返します。
func (w *cookieRewriteWriter) Header() http.Header {
	return w.hdr
}

// WriteHeader はステータスコードを設定します。
func (w *cookieRewriteWriter) WriteHeader(code int) {
	if w.wroteHdr {
		return
	}
	w.wroteHdr = true

	// Set-Cookie を書き換え
	w.flushHeadersWithRewrite()

	w.orig.WriteHeader(code)
}

// Write はボディを書き込みます。
func (w *cookieRewriteWriter) Write(p []byte) (int, error) {
	if !w.wroteHdr {
		w.WriteHeader(http.StatusOK)
	}
	return w.orig.Write(p)
}

// Flush はフラッシュを行います。
func (w *cookieRewriteWriter) Flush() {
	if f, ok := w.orig.(http.Flusher); ok {
		if !w.wroteHdr {
			w.WriteHeader(http.StatusOK)
		}
		f.Flush()
	}
}

// Hijack はコネクションをハイジャックします。
func (w *cookieRewriteWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	// WebSocket 等は WriteHeader を経由せずに Hijack されることがあるため、w.hdr 上で
	// Set-Cookie rewrite を確定させる。これは w.Header() を参照するタイプの Upgrade 実装にのみ
	// 効く防御であり、生バッファへ直書きする通常の hijack 経路では配線に出ない。
	if !w.wroteHdr {
		w.wroteHdr = true

		rawCookies := w.hdr.Values(headerSetCookie)
		if len(rawCookies) > 0 {
			w.hdr.Del(headerSetCookie)
			for _, raw := range rawCookies {
				w.hdr.Add(headerSetCookie, w.rewriteOrKeep(raw))
			}
		}
	}

	// WebSocket などで http.Hijacker を透過する
	h, ok := w.orig.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	return h.Hijack()
}

// Push は HTTP/2 Server Push を透過します。
func (w *cookieRewriteWriter) Push(target string, opts *http.PushOptions) error {
	p, ok := w.orig.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}
	return p.Push(target, opts)
}

// ReadFrom は io.Copy の最適化経路を透過します。
// wrapper 側でヘッダ書き換えを確定させてから orig に委譲します。
func (w *cookieRewriteWriter) ReadFrom(r io.Reader) (int64, error) {
	if !w.wroteHdr {
		w.WriteHeader(http.StatusOK)
	}

	if rf, ok := w.orig.(io.ReaderFrom); ok {
		return rf.ReadFrom(r)
	}
	// orig が ReaderFrom を持たない場合は素直にコピー（w には書かない：再帰回避）
	return io.Copy(w.orig, r)
}

// Unwrap は内側の ResponseWriter を返します。
// 他のミドルウェアや標準ライブラリが内側へアクセスしたい場合に役立ちます。
func (w *cookieRewriteWriter) Unwrap() http.ResponseWriter {
	return w.orig
}

// rewriteOrKeep は Set-Cookie を書き換え、失敗（空文字）時は元の raw を残します。
// 失敗時に消すと Cookie 消失になるため、この不変条件を1箇所に集約します。
func (w *cookieRewriteWriter) rewriteOrKeep(raw string) string {
	if r := w.cfg.RewriteSetCookie(raw); r != "" {
		return r
	}
	return raw
}

// flushHeadersWithRewrite はヘッダを書き込みます（Set-Cookie を書き換え）。
func (w *cookieRewriteWriter) flushHeadersWithRewrite() {
	for k, vv := range w.hdr {
		if strings.EqualFold(k, headerSetCookie) {
			continue
		}
		for _, v := range vv {
			w.orig.Header().Add(k, v)
		}
	}

	// Set-Cookie を rewrite してから追加
	for _, raw := range w.hdr.Values(headerSetCookie) {
		w.orig.Header().Add(headerSetCookie, w.rewriteOrKeep(raw))
	}
}
