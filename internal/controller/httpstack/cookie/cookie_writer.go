package cookie

import (
	"bufio"
	"io"
	"net"
	"net/http"
)

const headerSetCookie = "Set-Cookie"

type cookieRewriteWriter struct {
	orig http.ResponseWriter
	cfg  *SecurityCookie

	wroteHdr bool
}

// newCookieRewriteWriter は cookieRewriteWriter を構築します。
func newCookieRewriteWriter(orig http.ResponseWriter, cfg *SecurityCookie) *cookieRewriteWriter {
	return &cookieRewriteWriter{
		orig: orig,
		cfg:  cfg,
	}
}

// Header は元の ResponseWriter が持つヘッダをそのまま返します。
// 独自のヘッダを持たせると、ラップより前に書かれた値（X-Request-Id など）が読み出せなくなり、
// ワイヤ上のヘッダと、それを参照して組み立てるレスポンスボディとが食い違います。
func (w *cookieRewriteWriter) Header() http.Header {
	return w.orig.Header()
}

// WriteHeader はステータスコードを設定します。
func (w *cookieRewriteWriter) WriteHeader(code int) {
	if w.wroteHdr {
		return
	}
	w.wroteHdr = true

	w.rewriteSetCookies()

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
	// WebSocket 等は WriteHeader を経由せずに Hijack されることがあるため、ここで
	// Set-Cookie rewrite を確定させる。これは w.Header() を参照するタイプの Upgrade 実装にのみ
	// 効く防御であり、生バッファへ直書きする通常の hijack 経路では配線に出ない。
	if !w.wroteHdr {
		w.wroteHdr = true

		w.rewriteSetCookies()
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

// rewriteSetCookies は、ヘッダ上の Set-Cookie をセキュリティ属性の適用後の値へ置き換えます。
func (w *cookieRewriteWriter) rewriteSetCookies() {
	hdr := w.orig.Header()

	raws := hdr.Values(headerSetCookie)
	if len(raws) == 0 {
		return
	}

	hdr.Del(headerSetCookie)
	w.addRewrittenCookies(hdr, raws)
}

// addRewrittenCookies は、raws の各 Set-Cookie を書き換えて dst へ追加します。
func (w *cookieRewriteWriter) addRewrittenCookies(dst http.Header, raws []string) {
	for _, raw := range raws {
		dst.Add(headerSetCookie, w.rewriteOrKeep(raw))
	}
}
