package cookie

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
)

const headerSetCookie = "Set-Cookie"

type cookieRewriteWriter struct {
	orig http.ResponseWriter
	cfg  *SecurityCookie

	hdr        http.Header
	wroteHdr   bool
	statusCode int
}

// newCookieRewriteWriter は cookieRewriteWriter を構築します。
func newCookieRewriteWriter(orig http.ResponseWriter, cfg *SecurityCookie) *cookieRewriteWriter {
	return &cookieRewriteWriter{
		orig:       orig,
		cfg:        cfg,
		hdr:        make(http.Header),
		statusCode: http.StatusOK,
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
	w.statusCode = code

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
	// WebSocket 等は WriteHeader を経由せずに Hijack されることがあるため、
	// ここで Set-Cookie rewrite を “w.hdr 上で” 確定させる。
	// （Upgrade 実装が w.Header() の内容を参照してハンドシェイクを書き出すケースに備える）
	if !w.wroteHdr {
		w.wroteHdr = true

		rawCookies := w.hdr.Values(headerSetCookie)
		if len(rawCookies) > 0 {
			w.hdr.Del(headerSetCookie)
			for _, raw := range rawCookies {
				rewritten := w.cfg.RewriteSetCookie(raw)
				if rewritten == "" {
					// 失敗時に消すのは危険なので元を残す
					rewritten = raw
				}
				w.hdr.Add(headerSetCookie, rewritten)
			}
		}
	}

	// WebSocket などで http.Hijacker を透過する
	h, ok := w.orig.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("hijack not supported")
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
	rawCookies := w.hdr.Values(headerSetCookie)
	for _, raw := range rawCookies {
		rewritten := w.cfg.RewriteSetCookie(raw)
		if rewritten == "" {
			// 解析に失敗した等でも “消す” のは危険なので元を通す方が安全
			w.orig.Header().Add(headerSetCookie, raw)
			continue
		}
		w.orig.Header().Add(headerSetCookie, rewritten)
	}
}
