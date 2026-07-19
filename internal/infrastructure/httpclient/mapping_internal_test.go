package httpclient

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/pkg/xerrors"
)

// errRead は、readBody の read エラー分岐を検証するためのセンチネルです。
var errRead = xerrors.New("read failed")

// errReader は、常に read エラーを返す io.Reader です。
type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errRead }

func Test_Request_validate(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("必須項目が揃っていればnilを返す", func(t *testing.T) {
			t.Parallel()

			req := &Request{downstream: "sample", method: MethodGet(), url: "http://example.com"}

			require.NoError(t, req.validate())
		})

		t.Run("AllowRetryありでも冪等性キーがあればnilを返す", func(t *testing.T) {
			t.Parallel()

			req := &Request{
				downstream:     "sample",
				method:         MethodPost(),
				url:            "http://example.com",
				allowRetry:     true,
				idempotencyKey: "key-1",
			}

			require.NoError(t, req.validate())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Downstreamが空ならerrDownstreamRequiredを返す", func(t *testing.T) {
			t.Parallel()

			req := &Request{downstream: "", method: MethodGet(), url: "http://example.com"}

			require.ErrorIs(t, req.validate(), errDownstreamRequired)
		})

		t.Run("Methodがゼロ値ならerrMethodRequiredを返す", func(t *testing.T) {
			t.Parallel()

			req := &Request{downstream: "sample", method: Method{}, url: "http://example.com"}

			require.ErrorIs(t, req.validate(), errMethodRequired)
		})

		t.Run("AllowRetryありで冪等性キーが空ならerrIdempotencyKeyRequiredを返す", func(t *testing.T) {
			t.Parallel()

			req := &Request{
				downstream: "sample",
				method:     MethodPost(),
				url:        "http://example.com",
				allowRetry: true,
			}

			require.ErrorIs(t, req.validate(), errIdempotencyKeyRequired)
		})
	})
}

func Test_Header_clone(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("map と値スライスを深くコピーし元と共有しない", func(t *testing.T) {
			t.Parallel()

			original := Header{"X-Trace": {"a", "b"}}
			cloned := original.clone()

			require.Equal(t, original, cloned)

			// clone を書き換えても original に影響しない。
			cloned["X-Trace"][0] = "mutated"
			cloned["X-New"] = []string{"z"}
			assert.Equal(t, "a", original["X-Trace"][0])
			assert.NotContains(t, original, "X-New")
		})

		t.Run("nilヘッダはnilを返す", func(t *testing.T) {
			t.Parallel()

			var h Header

			assert.Nil(t, h.clone())
		})
	})
}

func Test_readBody(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("上限以内のボディを読み切って返す", func(t *testing.T) {
			t.Parallel()

			data, err := readBody(strings.NewReader("hello"), 10)

			require.NoError(t, err)
			assert.Equal(t, []byte("hello"), data)
		})

		t.Run("上限ちょうどのボディは読み切れる", func(t *testing.T) {
			t.Parallel()

			data, err := readBody(strings.NewReader("hello"), 5)

			require.NoError(t, err)
			assert.Equal(t, []byte("hello"), data)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("上限を超えるボディはerrResponseTooLargeを返す", func(t *testing.T) {
			t.Parallel()

			data, err := readBody(strings.NewReader("hello"), 4)

			require.ErrorIs(t, err, errResponseTooLarge)
			assert.Nil(t, data)
		})

		t.Run("read中のエラーをそのまま返す", func(t *testing.T) {
			t.Parallel()

			_, err := readBody(errReader{}, 10)

			require.ErrorIs(t, err, errRead)
		})
	})
}

func Test_redactURL(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("クエリ_userinfo_fragmentを除去しscheme_host_pathを残す", func(t *testing.T) {
			t.Parallel()

			out := redactURL("https://user:pass@api.example.com/rates?token=secret#frag")

			assert.Equal(t, "https://api.example.com/rates", out)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("パース不能なURLはupstreamを返す", func(t *testing.T) {
			t.Parallel()

			out := redactURL("://not a url\x7f")

			assert.Equal(t, "upstream", out)
		})
	})
}

func Test_staticRegistry_Profile(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("登録済みキーは対応するProfileを返す", func(t *testing.T) {
			t.Parallel()

			custom := DefaultProfile()
			custom.MaxAttempts = 99
			reg := NewRegistry(map[Downstream]Profile{"known": custom})

			got := reg.Profile("known")

			assert.Equal(t, 99, got.MaxAttempts)
		})

		t.Run("未登録キーはfallbackのDefaultProfileを返す", func(t *testing.T) {
			t.Parallel()

			reg := NewRegistry(map[Downstream]Profile{})

			got := reg.Profile("unknown")

			assert.Equal(t, DefaultProfile(), got)
		})
	})
}

// Test_client_attempt は、client.attempt を検証します。
func Test_client_attempt(t *testing.T) {
	t.Parallel()
	t.Skip("client.attempt は client_test.go の Test_client_Do_Send（2xx/4xx/5xx/transport失敗/URL不正/ボディ上限超過）で httptest サーバ経由の統合テストとして網羅されている")
}

// Test_client_doWithRetry は、client.doWithRetry を検証します。
func Test_client_doWithRetry(t *testing.T) {
	t.Parallel()
	t.Skip("client.doWithRetry は client_test.go の Test_client_Do_Retry/Backoff/Deadline/Breaker/Budget/MinimumAttempt で retry ループ全分岐を網羅している")
}

// Test_client_recordOutcome は、client.recordOutcome を検証します。
func Test_client_recordOutcome(t *testing.T) {
	t.Parallel()
	t.Skip("client.recordOutcome は client_test.go の Test_client_Do_Send（http_/circuit_open/canceled/transport の各 error class）で網羅されている")
}
