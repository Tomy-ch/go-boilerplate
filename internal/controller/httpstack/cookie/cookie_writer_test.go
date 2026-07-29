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

	// 透過（委譲）を検証するため、受け取った引数と返した値を記録します。
	pushedTarget string
	pushedOpts   *http.PushOptions
	hijackedConn net.Conn
	hijackedRW   *bufio.ReadWriter
}

// minimalResponseWriter は Hijacker/Pusher/ReaderFrom を実装しない最小ラッパーです。
type minimalResponseWriter struct{ rw http.ResponseWriter }

func newFakeOrig() *fakeOrig {
	return &fakeOrig{header: make(http.Header)}
}

func (f *fakeOrig) Header() http.Header         { return f.header }
func (f *fakeOrig) WriteHeader(code int)        { f.wroteCode = code }
func (f *fakeOrig) Write(p []byte) (int, error) { return f.body.Write(p) }
func (f *fakeOrig) Flush()                      { f.flushed = true }
func (f *fakeOrig) Push(target string, opts *http.PushOptions) error {
	f.pushed = true
	f.pushedTarget = target
	f.pushedOpts = opts
	return nil
}

func (f *fakeOrig) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	c1, c2 := net.Pipe()
	rw := bufio.NewReadWriter(bufio.NewReader(c2), bufio.NewWriter(c2))
	f.hijackedConn, f.hijackedRW = c1, rw
	return c1, rw, nil
}
func (f *fakeOrig) ReadFrom(r io.Reader) (int64, error) { return io.Copy(&f.body, r) }

func (n *minimalResponseWriter) Header() http.Header         { return n.rw.Header() }
func (n *minimalResponseWriter) Write(p []byte) (int, error) { return n.rw.Write(p) }
func (n *minimalResponseWriter) WriteHeader(code int)        { n.rw.WriteHeader(code) }

func Test_cookieRewriteWriter_Header(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("w.Header()への書き込みがorigのヘッダへ反映される", func(t *testing.T) {
			t.Parallel()
			orig := newFakeOrig()
			cfg := &SecurityCookie{}
			w := newCookieRewriteWriter(orig, cfg)

			w.Header().Add("X-Test", "v")

			assert.Equal(t, "v", orig.Header().Get("X-Test"))
		})

		t.Run("ラップ前にorigへ書かれたヘッダをw.Header()から読み出せる", func(t *testing.T) {
			t.Parallel()
			orig := newFakeOrig()
			orig.Header().Set("X-Request-Id", "rid-1")

			w := newCookieRewriteWriter(orig, &SecurityCookie{})

			assert.Equal(t, "rid-1", w.Header().Get("X-Request-Id"))
		})
	})
}

func Test_cookieRewriteWriter_WriteHeader(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Set-Cookieを書き換えてヘッダを転送する", func(t *testing.T) {
			t.Parallel()
			orig := newFakeOrig()
			b := true
			cfg := &SecurityCookie{applyToAll: true, forceHTTPOnly: &b}
			w := newCookieRewriteWriter(orig, cfg)

			w.Header().Add("Set-Cookie", "id=1; Path=/")
			w.Header().Add("X", "h")

			w.WriteHeader(http.StatusCreated)

			assert.Equal(t, http.StatusCreated, orig.wroteCode)
			assert.Equal(t, "h", orig.Header().Get("X"))
			sc := orig.Header()["Set-Cookie"]
			require.Len(t, sc, 1)
			assert.Contains(t, sc[0], "HttpOnly")
		})

		t.Run("既にヘッダ書き込み済みなら冪等に無視される", func(t *testing.T) {
			t.Parallel()
			orig := newFakeOrig()
			b := true
			cfg := &SecurityCookie{applyToAll: true, forceHTTPOnly: &b}
			w := newCookieRewriteWriter(orig, cfg)
			w.wroteHdr = true
			w.Header().Add("Set-Cookie", "id=1; Path=/")

			w.WriteHeader(http.StatusNoContent)
			assert.Equal(t, 0, orig.wroteCode)
			assert.Equal(t, []string{"id=1; Path=/"}, orig.Header().Values("Set-Cookie"))
		})
	})
}

func Test_cookieRewriteWriter_Write(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("WriteHeaderを呼んでボディを書き込む", func(t *testing.T) {
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

		t.Run("ヘッダ書き込み済みならWriteHeaderを再実行しない", func(t *testing.T) {
			t.Parallel()
			orig := newFakeOrig()
			cfg := &SecurityCookie{applyToAll: true}
			w := newCookieRewriteWriter(orig, cfg)
			w.wroteHdr = true

			n, err := w.Write([]byte("hi"))
			require.NoError(t, err)
			assert.Equal(t, 2, n)
			assert.Equal(t, 0, orig.wroteCode)
			assert.Equal(t, "hi", orig.body.String())
		})

		t.Run("ボディ書き込みの前にSet-Cookieが書き換えられる", func(t *testing.T) {
			t.Parallel()
			orig := newFakeOrig()
			b := true
			cfg := &SecurityCookie{applyToAll: true, forceHTTPOnly: &b}
			w := newCookieRewriteWriter(orig, cfg)

			w.Header().Add("Set-Cookie", "id=1; Path=/")

			_, err := w.Write([]byte("hello"))
			require.NoError(t, err)

			sc := orig.Header().Values("Set-Cookie")
			require.Len(t, sc, 1)
			assert.Contains(t, sc[0], "HttpOnly")
		})
	})
}

func Test_cookieRewriteWriter_Flush(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Flusherへの委譲時にWriteHeaderを経由する", func(t *testing.T) {
			t.Parallel()
			orig := newFakeOrig()
			cfg := &SecurityCookie{}
			w := newCookieRewriteWriter(orig, cfg)

			w.Flush()
			assert.True(t, orig.flushed)
			assert.Equal(t, http.StatusOK, orig.wroteCode)
		})

		t.Run("origがFlusherを実装しない場合はno-op", func(t *testing.T) {
			t.Parallel()
			orig := newFakeOrig()
			w := newCookieRewriteWriter(&minimalResponseWriter{rw: orig}, &SecurityCookie{})

			w.Flush()
			assert.False(t, orig.flushed)
			assert.Equal(t, 0, orig.wroteCode)
		})

		t.Run("フラッシュの前にSet-Cookieが書き換えられる", func(t *testing.T) {
			t.Parallel()
			orig := newFakeOrig()
			b := true
			cfg := &SecurityCookie{applyToAll: true, forceHTTPOnly: &b}
			w := newCookieRewriteWriter(orig, cfg)

			w.Header().Add("Set-Cookie", "id=1; Path=/")

			w.Flush()

			sc := orig.Header().Values("Set-Cookie")
			require.Len(t, sc, 1)
			assert.Contains(t, sc[0], "HttpOnly")
		})
	})
}

func Test_cookieRewriteWriter_Hijack(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("origがHijackerを実装している場合は書き換えて透過する", func(t *testing.T) {
			t.Parallel()
			orig := newFakeOrig()
			b := true
			cfg := &SecurityCookie{applyToAll: true, forceHTTPOnly: &b}
			w := newCookieRewriteWriter(orig, cfg)

			w.Header().Add("Set-Cookie", "id=1")
			conn, rw, err := w.Hijack()
			require.NoError(t, err)
			require.NotNil(t, conn)
			require.NotNil(t, rw)
			assert.Same(t, orig.hijackedConn, conn)
			assert.Same(t, orig.hijackedRW, rw)
			assert.True(t, w.wroteHdr)
			sc := orig.Header().Values("Set-Cookie")
			require.Len(t, sc, 1)
			assert.Contains(t, sc[0], "HttpOnly")
			require.NoError(t, conn.Close())
		})

		t.Run("既にwroteHdrがtrueの場合はrewriteをスキップする", func(t *testing.T) {
			t.Parallel()
			orig := newFakeOrig()
			b := true
			cfg := &SecurityCookie{applyToAll: true, forceHTTPOnly: &b}
			w := newCookieRewriteWriter(orig, cfg)
			w.Header().Add("Set-Cookie", "id=1")
			w.wroteHdr = true
			conn, rw, err := w.Hijack()
			require.NoError(t, err)
			require.NotNil(t, conn)
			require.NotNil(t, rw)
			assert.Equal(t, []string{"id=1"}, orig.Header().Values("Set-Cookie"))
			require.NoError(t, conn.Close())
		})

		t.Run("Rewriteが失敗したら元のrawをhdrに残す", func(t *testing.T) {
			t.Parallel()
			orig := newFakeOrig()
			cfg := &SecurityCookie{applyToAll: true}
			w := newCookieRewriteWriter(orig, cfg)
			// 不正な Cookie（'=' 無し）なので RewriteSetCookie は "" を返す
			w.Header().Add("Set-Cookie", "NoEquals")

			conn, rw, err := w.Hijack()
			require.NoError(t, err)
			require.NotNil(t, conn)
			require.NotNil(t, rw)
			assert.Equal(t, []string{"NoEquals"}, orig.Header().Values("Set-Cookie"))
			require.NoError(t, conn.Close())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("origがHijackerを実装していない場合は書き換えを確定させてからErrNotSupportedを返す", func(t *testing.T) {
			t.Parallel()
			wrapper := &minimalResponseWriter{rw: newFakeOrig()}
			var orig http.ResponseWriter = wrapper
			b := true
			w := newCookieRewriteWriter(orig, &SecurityCookie{applyToAll: true, forceHTTPOnly: &b})

			w.Header().Add("Set-Cookie", "id=1; Path=/")

			_, _, err := w.Hijack()
			require.ErrorIs(t, err, http.ErrNotSupported)

			sc := wrapper.Header().Values("Set-Cookie")
			require.Len(t, sc, 1)
			assert.Contains(t, sc[0], "HttpOnly")
		})
	})
}

func Test_cookieRewriteWriter_Push(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("origがPusherを実装している場合は引数をそのまま渡して透過する", func(t *testing.T) {
			t.Parallel()
			orig := newFakeOrig()
			cfg := &SecurityCookie{applyToAll: true}
			w := newCookieRewriteWriter(orig, cfg)
			opts := &http.PushOptions{Method: http.MethodGet}

			err := w.Push("/a", opts)
			require.NoError(t, err)
			assert.True(t, orig.pushed)
			assert.Equal(t, "/a", orig.pushedTarget)
			assert.Same(t, opts, orig.pushedOpts)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("origがPusherを実装していない場合はErrNotSupportedを返す", func(t *testing.T) {
			t.Parallel()
			wrapper := &minimalResponseWriter{rw: newFakeOrig()}
			var orig http.ResponseWriter = wrapper
			w := newCookieRewriteWriter(orig, &SecurityCookie{})
			err := w.Push("/p", &http.PushOptions{})
			require.ErrorIs(t, err, http.ErrNotSupported)
		})
	})
}

func Test_cookieRewriteWriter_ReadFrom(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("origがReaderFromを実装している場合は透過して読み取る", func(t *testing.T) {
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

		t.Run("origがReaderFromを実装していない場合はio.Copy経由で書き込まれる", func(t *testing.T) {
			t.Parallel()
			wrapper := &minimalResponseWriter{rw: newFakeOrig()}
			var orig http.ResponseWriter = wrapper
			w := newCookieRewriteWriter(orig, &SecurityCookie{applyToAll: true})

			r := strings.NewReader("xyz")
			n, err := w.ReadFrom(r)
			require.NoError(t, err)
			assert.Equal(t, int64(3), n)
			f, ok := wrapper.rw.(*fakeOrig)
			require.True(t, ok)
			assert.Equal(t, "xyz", f.body.String())
		})

		t.Run("読み取りの前にSet-Cookieが書き換えられる", func(t *testing.T) {
			t.Parallel()
			orig := newFakeOrig()
			b := true
			cfg := &SecurityCookie{applyToAll: true, forceHTTPOnly: &b}
			w := newCookieRewriteWriter(orig, cfg)

			w.Header().Add("Set-Cookie", "id=1; Path=/")

			_, err := w.ReadFrom(strings.NewReader("payload"))
			require.NoError(t, err)

			sc := orig.Header().Values("Set-Cookie")
			require.Len(t, sc, 1)
			assert.Contains(t, sc[0], "HttpOnly")
		})
	})
}

func Test_cookieRewriteWriter_Unwrap(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("内側のResponseWriterを返す", func(t *testing.T) {
			t.Parallel()
			orig := newFakeOrig()
			cfg := &SecurityCookie{}
			w := newCookieRewriteWriter(orig, cfg)
			assert.Equal(t, orig, w.Unwrap())
		})
	})
}

func Test_cookieRewriteWriter_rewriteOrKeep(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("書き換え成功時は書き換え後の値を返す", func(t *testing.T) {
			t.Parallel()
			b := true
			cfg := &SecurityCookie{applyToAll: true, forceHTTPOnly: &b}
			w := newCookieRewriteWriter(newFakeOrig(), cfg)

			got := w.rewriteOrKeep("id=1; Path=/")
			assert.Contains(t, got, "HttpOnly")
		})

		t.Run("書き換え失敗（空文字）時は元のrawを残す", func(t *testing.T) {
			t.Parallel()
			cfg := &SecurityCookie{applyToAll: true}
			w := newCookieRewriteWriter(newFakeOrig(), cfg)

			// 不正な Cookie（'=' 無し）なので RewriteSetCookie は "" を返す
			assert.Equal(t, "NoEquals", w.rewriteOrKeep("NoEquals"))
		})
	})
}

func Test_cookieRewriteWriter_addRewrittenCookies(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("全てのSet-Cookieを書き換えて指定ヘッダへ追加する", func(t *testing.T) {
			t.Parallel()
			b := true
			cfg := &SecurityCookie{applyToAll: true, forceHTTPOnly: &b}
			w := newCookieRewriteWriter(newFakeOrig(), cfg)

			dst := make(http.Header)
			w.addRewrittenCookies(dst, []string{"a=1; Path=/", "b=2; Path=/"})

			got := dst.Values(headerSetCookie)
			require.Len(t, got, 2)
			assert.Contains(t, got[0], "a=1")
			assert.Contains(t, got[0], "HttpOnly")
			assert.Contains(t, got[1], "b=2")
			assert.Contains(t, got[1], "HttpOnly")
		})

		t.Run("書き換え不能なSet-Cookieも欠落させず元のまま追加する", func(t *testing.T) {
			t.Parallel()
			cfg := &SecurityCookie{applyToAll: true}
			w := newCookieRewriteWriter(newFakeOrig(), cfg)

			dst := make(http.Header)
			w.addRewrittenCookies(dst, []string{"NoEquals"})

			assert.Equal(t, []string{"NoEquals"}, dst.Values(headerSetCookie))
		})

		t.Run("空のSet-Cookie列では何も追加しない", func(t *testing.T) {
			t.Parallel()
			cfg := &SecurityCookie{applyToAll: true}
			w := newCookieRewriteWriter(newFakeOrig(), cfg)

			dst := make(http.Header)
			w.addRewrittenCookies(dst, nil)

			assert.Empty(t, dst.Values(headerSetCookie))
		})

		t.Run("既存のSet-Cookieを保持したまま追記する", func(t *testing.T) {
			t.Parallel()
			cfg := &SecurityCookie{applyToAll: true}
			w := newCookieRewriteWriter(newFakeOrig(), cfg)

			dst := make(http.Header)
			dst.Add(headerSetCookie, "existing=0; Path=/")
			w.addRewrittenCookies(dst, []string{"a=1; Path=/"})

			got := dst.Values(headerSetCookie)
			require.Len(t, got, 2)
			assert.Contains(t, got[0], "existing=0")
			assert.Contains(t, got[1], "a=1")
		})
	})
}

func Test_cookieRewriteWriter_rewriteSetCookies(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Set-Cookieを書き換えSet-Cookie以外のヘッダはそのまま残す", func(t *testing.T) {
			t.Parallel()
			orig := newFakeOrig()
			b := true
			cfg := &SecurityCookie{applyToAll: true, forceSecure: &b}
			w := newCookieRewriteWriter(orig, cfg)

			w.Header().Add("X-Test", "v")
			w.Header().Add("Set-Cookie", "id=1; Path=/")

			w.rewriteSetCookies()

			assert.Equal(t, "v", orig.Header().Get("X-Test"))
			sc := orig.Header()["Set-Cookie"]
			require.Len(t, sc, 1)
			assert.Contains(t, sc[0], "Secure")
		})

		t.Run("複数のSet-Cookieを重複させず全て書き換える", func(t *testing.T) {
			t.Parallel()
			orig := newFakeOrig()
			b := true
			cfg := &SecurityCookie{applyToAll: true, forceSecure: &b}
			w := newCookieRewriteWriter(orig, cfg)

			w.Header().Add("Set-Cookie", "a=1; Path=/")
			w.Header().Add("Set-Cookie", "b=2; Path=/")

			w.rewriteSetCookies()

			sc := orig.Header()["Set-Cookie"]
			require.Len(t, sc, 2)
			assert.Contains(t, sc[0], "a=1")
			assert.Contains(t, sc[0], "Secure")
			assert.Contains(t, sc[1], "b=2")
			assert.Contains(t, sc[1], "Secure")
		})

		t.Run("Set-Cookieが無ければヘッダを変更しない", func(t *testing.T) {
			t.Parallel()
			orig := newFakeOrig()
			cfg := &SecurityCookie{applyToAll: true}
			w := newCookieRewriteWriter(orig, cfg)

			w.Header().Add("X-Test", "v")

			w.rewriteSetCookies()

			assert.Equal(t, "v", orig.Header().Get("X-Test"))
			assert.Empty(t, orig.Header().Values("Set-Cookie"))
		})

		t.Run("Rewriteに失敗したら元のrawを使う", func(t *testing.T) {
			t.Parallel()
			orig := newFakeOrig()
			cfg := &SecurityCookie{applyToAll: true}
			w := newCookieRewriteWriter(orig, cfg)

			// 不正な Cookie（'=' 無し）なので RewriteSetCookie は "" を返す
			w.Header().Add("Set-Cookie", "NoEquals")
			w.rewriteSetCookies()

			sc := orig.Header()["Set-Cookie"]
			assert.Equal(t, []string{"NoEquals"}, sc)
		})
	})
}

func Test_newCookieRewriteWriter(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("origとcfgを保持しヘッダ未書き込みの状態で構築する", func(t *testing.T) {
			t.Parallel()
			orig := newFakeOrig()
			cfg := &SecurityCookie{applyToAll: true}

			w := newCookieRewriteWriter(orig, cfg)

			assert.Same(t, orig, w.orig)
			assert.Same(t, cfg, w.cfg)
			assert.False(t, w.wroteHdr)
		})
	})
}
