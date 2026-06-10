package cookie

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeOrig はテスト用の簡易 ResponseWriter 実装です。
type fakeOrig struct {
	header    http.Header
	wroteCode int
	body      bytes.Buffer
	flushed   bool
	pushed    bool
}

// minimalResponseWriter は Hijacker/Pusher/ReaderFrom を実装しない最小ラッパーです。
type minimalResponseWriter struct{ rw http.ResponseWriter }

func newFakeOrig() *fakeOrig {
	return &fakeOrig{header: make(http.Header)}
}

func (f *fakeOrig) Header() http.Header                      { return f.header }
func (f *fakeOrig) WriteHeader(code int)                     { f.wroteCode = code }
func (f *fakeOrig) Write(p []byte) (int, error)              { return f.body.Write(p) }
func (f *fakeOrig) Flush()                                   { f.flushed = true }
func (f *fakeOrig) Push(_ string, _ *http.PushOptions) error { f.pushed = true; return nil }
func (f *fakeOrig) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	c1, c2 := net.Pipe()
	rw := bufio.NewReadWriter(bufio.NewReader(c2), bufio.NewWriter(c2))
	return c1, rw, nil
}
func (f *fakeOrig) ReadFrom(r io.Reader) (int64, error) { return io.Copy(&f.body, r) }

func (n *minimalResponseWriter) Header() http.Header         { return n.rw.Header() }
func (n *minimalResponseWriter) Write(p []byte) (int, error) { return n.rw.Write(p) }
func (n *minimalResponseWriter) WriteHeader(code int)        { n.rw.WriteHeader(code) }

func Test_cookieRewriteWriter_Header(t *testing.T) {
	t.Parallel()
	t.Run("Header: w.Header() で内部ヘッダに値を設定できる", func(t *testing.T) {
		t.Parallel()
		orig := newFakeOrig()
		cfg := &SecurityCookie{}
		w := newCookieRewriteWriter(orig, cfg)

		h := w.Header()
		h.Add("X-Test", "v")
		assert.Equal(t, "v", w.hdr.Get("X-Test"))
	})
}

func Test_cookieRewriteWriter_WriteHeader(t *testing.T) {
	t.Parallel()
	t.Run("WriteHeader: Set-Cookie を書き換えてヘッダを転送する", func(t *testing.T) {
		t.Parallel()
		orig := newFakeOrig()
		// cfg で HttpOnly を強制する
		b := true
		cfg := &SecurityCookie{applyToAll: true, forceHTTPOnly: &b}
		w := newCookieRewriteWriter(orig, cfg)

		w.Header().Add("Set-Cookie", "id=1; Path=/")
		w.Header().Add("X", "h")

		w.WriteHeader(http.StatusCreated)

		assert.Equal(t, 201, orig.wroteCode)
		assert.Equal(t, "h", orig.Header().Get("X"))
		sc := orig.Header()["Set-Cookie"]
		assert.Len(t, sc, 1)
		assert.Contains(t, sc[0], "HttpOnly")
	})

	t.Run("WriteHeader: 既にヘッダ書き込み済みなら無視される", func(t *testing.T) {
		t.Parallel()
		orig2 := newFakeOrig()
		cfg2 := &SecurityCookie{applyToAll: true}
		w2 := newCookieRewriteWriter(orig2, cfg2)
		// 先にフラグを立てる
		w2.wroteHdr = true
		w2.hdr.Add("X-Should-Not", "v")
		// orig2 の状態は変わらないはず
		w2.WriteHeader(http.StatusNoContent)
		assert.Equal(t, 0, orig2.wroteCode)
		assert.Empty(t, orig2.Header().Get("X-Should-Not"))
	})
}

func Test_cookieRewriteWriter_Write(t *testing.T) {
	t.Parallel()
	t.Run("Write: WriteHeader を呼んでボディを書き込む", func(t *testing.T) {
		t.Parallel()
		orig := newFakeOrig()
		cfg := &SecurityCookie{applyToAll: true}
		w := newCookieRewriteWriter(orig, cfg)

		n, err := w.Write([]byte("hello"))
		require.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, http.StatusOK, orig.wroteCode)
		assert.Equal(t, "hello", orig.body.String())
	})
}

func Test_cookieRewriteWriter_Flush(t *testing.T) {
	t.Parallel()
	t.Run("Flush: Flusher を呼び WriteHeader を経由する", func(t *testing.T) {
		t.Parallel()
		orig := newFakeOrig()
		cfg := &SecurityCookie{}
		w := newCookieRewriteWriter(orig, cfg)

		w.Flush()
		assert.True(t, orig.flushed)
		assert.Equal(t, http.StatusOK, orig.wroteCode)
	})
}

func Test_cookieRewriteWriter_Hijack(t *testing.T) {
	t.Parallel()
	t.Run("Hijack: orig が Hijacker を実装している場合は書き換えて透過される", func(t *testing.T) {
		t.Parallel()
		orig := newFakeOrig()
		b := true
		cfg := &SecurityCookie{applyToAll: true, forceHTTPOnly: &b}
		w := newCookieRewriteWriter(orig, cfg)

		w.hdr.Add("Set-Cookie", "id=1")
		conn, rw, err := w.Hijack()
		require.NoError(t, err)
		require.NotNil(t, conn)
		require.NotNil(t, rw)
		// hijack 確定で wroteHdr が立っている
		assert.True(t, w.wroteHdr)
		// header 上で書き換え済みの Set-Cookie（HttpOnly 付与）が存在する
		sc := w.hdr.Values("Set-Cookie")
		require.Len(t, sc, 1)
		assert.Contains(t, sc[0], "HttpOnly")
		err = conn.Close()
		require.NoError(t, err)
	})

	t.Run("Hijack: orig が Hijacker を実装していない場合はエラー", func(t *testing.T) {
		t.Parallel()
		wrapper := &minimalResponseWriter{rw: newFakeOrig()}
		var orig http.ResponseWriter = wrapper
		w := newCookieRewriteWriter(orig, &SecurityCookie{})
		_, _, err := w.Hijack()
		require.ErrorIs(t, err, http.ErrNotSupported)
	})

	t.Run("Hijack: 既に wroteHdr が true の場合は rewrite をスキップする", func(t *testing.T) {
		t.Parallel()
		orig := newFakeOrig()
		b := true
		cfg := &SecurityCookie{applyToAll: true, forceHTTPOnly: &b}
		w := newCookieRewriteWriter(orig, cfg)
		// 正常な Cookie を入れ、wroteHdr を先に立てる
		w.hdr.Add("Set-Cookie", "id=1")
		w.wroteHdr = true
		conn, rw, err := w.Hijack()
		require.NoError(t, err)
		require.NotNil(t, conn)
		require.NotNil(t, rw)
		// rewrite がスキップされ HttpOnly は付与されない（ガード除去時は付与され FAIL する）
		assert.Equal(t, []string{"id=1"}, w.hdr.Values("Set-Cookie"))
		err = conn.Close()
		require.NoError(t, err)
	})

	t.Run("Hijack: Rewrite が失敗したら元の raw を hdr に残す", func(t *testing.T) {
		t.Parallel()
		orig := newFakeOrig()
		cfg := &SecurityCookie{applyToAll: true}
		w := newCookieRewriteWriter(orig, cfg)
		// 不正な Cookie（'=' 無し）なので RewriteSetCookie は "" を返す
		w.hdr.Add("Set-Cookie", "NoEquals")

		conn, rw, err := w.Hijack()
		require.NoError(t, err)
		require.NotNil(t, conn)
		require.NotNil(t, rw)
		// hdr に元の raw が残っている
		assert.Equal(t, []string{"NoEquals"}, w.hdr.Values("Set-Cookie"))
		err = conn.Close()
		require.NoError(t, err)
	})
}

func Test_cookieRewriteWriter_Push(t *testing.T) {
	t.Parallel()
	t.Run("Push: orig が Pusher を実装している場合は透過する", func(t *testing.T) {
		t.Parallel()
		orig := newFakeOrig()
		cfg := &SecurityCookie{applyToAll: true}
		w := newCookieRewriteWriter(orig, cfg)

		// orig は Pusher を実装しているので成功し委譲される
		err := w.Push("/a", &http.PushOptions{})
		require.NoError(t, err)
		assert.True(t, orig.pushed)
	})

	t.Run("Push: orig が Pusher を実装していない場合は ErrNotSupported を返す", func(t *testing.T) {
		t.Parallel()
		wrapper := &minimalResponseWriter{rw: newFakeOrig()}
		var orig http.ResponseWriter = wrapper
		w := newCookieRewriteWriter(orig, &SecurityCookie{})
		err := w.Push("/p", &http.PushOptions{})
		require.ErrorIs(t, err, http.ErrNotSupported)
	})
}

func Test_cookieRewriteWriter_ReadFrom(t *testing.T) {
	t.Parallel()
	t.Run("ReadFrom: orig が ReaderFrom を実装している場合は透過して読み取る", func(t *testing.T) {
		t.Parallel()
		orig := newFakeOrig()
		cfg := &SecurityCookie{applyToAll: true}
		w := newCookieRewriteWriter(orig, cfg)

		r := strings.NewReader("payload")
		n, err := w.ReadFrom(r)
		require.NoError(t, err)
		assert.Equal(t, int64(7), n)
		assert.Equal(t, "payload", orig.body.String())
	})

	t.Run("ReadFrom: orig が ReaderFrom を実装していない場合は io.Copy 経由で書き込まれる", func(t *testing.T) {
		t.Parallel()
		wrapper := &minimalResponseWriter{rw: newFakeOrig()}
		var orig http.ResponseWriter = wrapper
		w := newCookieRewriteWriter(orig, &SecurityCookie{applyToAll: true})

		r := strings.NewReader("xyz")
		n, err := w.ReadFrom(r)
		require.NoError(t, err)
		assert.Positive(t, n)
		f, ok := wrapper.rw.(*fakeOrig)
		require.True(t, ok)
		assert.Equal(t, "xyz", f.body.String())
	})
}

func Test_cookieRewriteWriter_Unwrap(t *testing.T) {
	t.Parallel()
	t.Run("Unwrap: 内側の ResponseWriter を返す", func(t *testing.T) {
		t.Parallel()
		orig := newFakeOrig()
		cfg := &SecurityCookie{}
		w := newCookieRewriteWriter(orig, cfg)
		assert.Equal(t, orig, w.Unwrap())
	})
}

func Test_cookieRewriteWriter_flushHeadersWithRewrite(t *testing.T) {
	t.Parallel()
	t.Run("flushHeadersWithRewrite: Set-Cookie を書き換えて元ヘッダを転送する（Secure 付与）", func(t *testing.T) {
		t.Parallel()
		orig := newFakeOrig()
		// cfg で Secure を強制する
		b := true
		cfg := &SecurityCookie{applyToAll: true, forceSecure: &b}
		w := newCookieRewriteWriter(orig, cfg)

		w.hdr.Add("X-Test", "v")
		w.hdr.Add("Set-Cookie", "id=1; Path=/")

		w.flushHeadersWithRewrite()

		// Set-Cookie 以外のヘッダは転送される
		assert.Equal(t, "v", orig.Header().Get("X-Test"))
		// Set-Cookie は書き換えられて追加される
		sc := orig.Header()["Set-Cookie"]
		assert.Len(t, sc, 1)
		assert.Contains(t, sc[0], "Secure")
	})

	t.Run("flushHeadersWithRewrite: Rewrite に失敗したら元の raw を使う", func(t *testing.T) {
		t.Parallel()
		orig := newFakeOrig()
		cfg := &SecurityCookie{applyToAll: true}
		w := newCookieRewriteWriter(orig, cfg)

		// 不正な Cookie（'=' 無し）なので RewriteSetCookie は "" を返す
		w.hdr.Add("Set-Cookie", "NoEquals")
		w.flushHeadersWithRewrite()

		sc := orig.Header()["Set-Cookie"]
		assert.Equal(t, []string{"NoEquals"}, sc)
	})
}
