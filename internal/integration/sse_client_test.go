package integration

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/controller/stream/gen"
)

// sseReadTimeout は、フレームが 1 つ届くのを待つ上限です。超えたら実装が動いていないので落とします。
const sseReadTimeout = 3 * time.Second

// sseFrame は、SSE の 1 フレームです。business event だけが id を持ち、control と heartbeat は持ちません。
type sseFrame struct {
	id      string
	event   string
	data    string
	comment bool
}

// sseClient は、docs/design/realtime-delivery.md §4.3 の client 契約を実装する**テスト専用**の
// 参照 client です。出荷する SDK ではなく、server がその契約を前提にしてよいことを固定するために置きます。
//
// 契約のうちここで実装するのは 2 点です。Last-Event-ID は business event の id だけで進むこと、
// そして STOP / REAUTHENTICATE / RESYNC を受けたら server の EOF を待たず自分から接続を閉じること
// （閉じないと EventSource が自動再接続し、再接続の loop になります）。
type sseClient struct {
	t   *testing.T
	res *http.Response
	r   *bufio.Reader

	// lastEventID は、次に繋ぎ直すときに提示する位置です。control では動きません。
	lastEventID string
	// lastControl は、最後に受け取った制御指示です。
	lastControl *gen.ControlEvent
	// closedBySelf は、指示を受けて自分から閉じたことを表します。
	closedBySelf bool

	closeOnce sync.Once
}

// connectSSE は、path へ 1 本繋ぎます。SSE を読み続けるため client 側に timeout は設けません。
// 200 以外で返ることもあるので、レスポンスは呼び出し側が確かめます。
func connectSSE(t *testing.T, srv *Server, path string) (*sseClient, *http.Response) {
	t.Helper()

	return connectSSEWithHeaders(t, srv, path, nil)
}

// connectSSEWithHeaders は、追加のヘッダ（再接続の Last-Event-ID など）を付けて 1 本繋ぎます。
func connectSSEWithHeaders(
	t *testing.T, srv *Server, path string, headers http.Header,
) (*sseClient, *http.Response) {
	t.Helper()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.baseURL+path, nil)
	require.NoError(t, err)

	for k, vs := range headers {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}

	res, err := (&http.Client{}).Do(req)
	require.NoError(t, err)

	c := &sseClient{t: t, res: res, r: bufio.NewReader(res.Body)}
	t.Cleanup(c.close)

	return c, res
}

// close は、接続を閉じます。2 度目以降は何もしません。
func (c *sseClient) close() {
	c.closeOnce.Do(func() { _ = c.res.Body.Close() })
}

// next は、次のフレームを 1 つ受け取り、契約どおりに状態を進めます。届かなければテストを落とします。
func (c *sseClient) next() sseFrame {
	c.t.Helper()

	type result struct {
		frame sseFrame
		err   error
	}
	ch := make(chan result, 1)
	go func() {
		f, err := c.readFrame()
		ch <- result{frame: f, err: err}
	}()

	select {
	case got := <-ch:
		require.NoError(c.t, got.err)
		c.apply(got.frame)

		return got.frame
	case <-time.After(sseReadTimeout):
		c.t.Fatal("SSE のフレームが届かなかった")

		return sseFrame{}
	}
}

// apply は、受け取ったフレームを契約に沿って処理します。
func (c *sseClient) apply(f sseFrame) {
	c.t.Helper()

	if f.id != "" {
		c.lastEventID = f.id
	}

	if f.event != "control" {
		return
	}

	var ev gen.ControlEvent
	require.NoError(c.t, json.Unmarshal([]byte(f.data), &ev))
	c.lastControl = &ev

	if mustCloseOnControl(ev.Action) {
		c.closedBySelf = true
		c.close()
	}
}

// readFrame は、空行までを 1 フレームとして読みます。
func (c *sseClient) readFrame() (sseFrame, error) {
	var f sseFrame

	for {
		line, err := c.r.ReadString('\n')
		if err != nil {
			return f, err
		}

		line = strings.TrimSuffix(line, "\n")
		switch {
		case line == "":
			return f, nil
		case strings.HasPrefix(line, ":"):
			f.comment = true
		case strings.HasPrefix(line, "id: "):
			f.id = strings.TrimPrefix(line, "id: ")
		case strings.HasPrefix(line, "event: "):
			f.event = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: "):
			f.data = strings.TrimPrefix(line, "data: ")
		}
	}
}

// mustCloseOnControl は、client が server の EOF を待たず自ら閉じなければならない指示かを返します。
func mustCloseOnControl(a gen.ControlEventAction) bool {
	return a == gen.STOP || a == gen.REAUTHENTICATE || a == gen.RESYNC
}
