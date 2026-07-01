package httpclient_test

import (
	"testing"

	"go-boilerplate/internal/infrastructure/httpclient"

	"github.com/stretchr/testify/assert"
)

func TestMethod(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("GETは文字列GETを表す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "GET", httpclient.MethodGet().String())
		})

		t.Run("POSTは文字列POSTを表す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "POST", httpclient.MethodPost().String())
		})

		t.Run("PUTは文字列PUTを表す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "PUT", httpclient.MethodPut().String())
		})

		t.Run("PATCHは文字列PATCHを表す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "PATCH", httpclient.MethodPatch().String())
		})

		t.Run("DELETEは文字列DELETEを表す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "DELETE", httpclient.MethodDelete().String())
		})

		t.Run("ゼロ値は空文字を表す", func(t *testing.T) {
			t.Parallel()
			assert.Empty(t, httpclient.Method{}.String())
		})
	})
}

func TestRequest(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("各フィールドを保持できる", func(t *testing.T) {
			t.Parallel()

			req := httpclient.NewRequest(
				httpclient.MethodPost(), "payment", "https://example.com/charge",
				httpclient.WithHeader(httpclient.Header{"Content-Type": {"application/json"}}),
				httpclient.WithBody([]byte(`{"amount":100}`)),
				httpclient.WithRetry("key-1"),
			)

			assert.Equal(t, httpclient.Downstream("payment"), req.Downstream())
			assert.Equal(t, httpclient.MethodPost(), req.Method())
			assert.Equal(t, "https://example.com/charge", req.URL())
			assert.Equal(t, []string{"application/json"}, req.Header()["Content-Type"])
			assert.Equal(t, []byte(`{"amount":100}`), req.Body())
			assert.Equal(t, "key-1", req.IdempotencyKey())
			assert.True(t, req.AllowRetry())
		})

		t.Run("HeaderとBodyのゲッターは防御的コピーを返し内部状態は不変", func(t *testing.T) {
			t.Parallel()

			req := httpclient.NewRequest(
				httpclient.MethodPost(), "payment", "https://example.com/charge",
				httpclient.WithHeader(httpclient.Header{"X-Token": {"secret"}}),
				httpclient.WithBody([]byte("original")),
			)

			// 返り値を破壊的に変更しても、Request 内部には影響しないこと。
			got := req.Header()
			got["X-Token"][0] = "tampered"
			got["X-Injected"] = []string{"x"}
			body := req.Body()
			body[0] = 'X'

			assert.Equal(t, []string{"secret"}, req.Header()["X-Token"], "Header の値スライスが共有されていないこと")
			assert.NotContains(t, req.Header(), "X-Injected", "Header の map が共有されていないこと")
			assert.Equal(t, []byte("original"), req.Body(), "Body スライスが共有されていないこと")
		})
	})
}

func TestNewRequest(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("必須項目のみで生成し任意項目は未設定になる", func(t *testing.T) {
			t.Parallel()

			req := httpclient.NewRequest(httpclient.MethodGet(), "rates", "https://example.com/rates")

			assert.Equal(t, httpclient.MethodGet(), req.Method())
			assert.Equal(t, httpclient.Downstream("rates"), req.Downstream())
			assert.Equal(t, "https://example.com/rates", req.URL())
			assert.Nil(t, req.Header())
			assert.Nil(t, req.Body())
			assert.Empty(t, req.IdempotencyKey())
			assert.False(t, req.AllowRetry())
		})

		t.Run("オプションでheader_body_idempotencyKeyを設定できる", func(t *testing.T) {
			t.Parallel()

			header := httpclient.Header{"Content-Type": {"application/json"}}
			req := httpclient.NewRequest(httpclient.MethodPost(), "pay", "https://example.com/charge",
				httpclient.WithHeader(header),
				httpclient.WithBody([]byte("{}")),
				httpclient.WithIdempotencyKey("key-1"),
			)

			assert.Equal(t, header, req.Header())
			assert.Equal(t, []byte("{}"), req.Body())
			assert.Equal(t, "key-1", req.IdempotencyKey())
			assert.False(t, req.AllowRetry(), "WithIdempotencyKey は retry を許可しない")
		})

		t.Run("WithRetryはAllowRetryとIdempotencyKeyを同時に設定する", func(t *testing.T) {
			t.Parallel()

			req := httpclient.NewRequest(httpclient.MethodPost(), "pay", "https://example.com/charge",
				httpclient.WithRetry("key-2"),
			)

			assert.True(t, req.AllowRetry())
			assert.Equal(t, "key-2", req.IdempotencyKey())
		})
	})
}
