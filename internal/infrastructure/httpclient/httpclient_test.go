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

		cases := map[string]struct {
			method httpclient.Method
			want   string
		}{
			"GETは文字列GETを表す":       {method: httpclient.MethodGet, want: "GET"},
			"POSTは文字列POSTを表す":     {method: httpclient.MethodPost, want: "POST"},
			"PUTは文字列PUTを表す":       {method: httpclient.MethodPut, want: "PUT"},
			"PATCHは文字列PATCHを表す":   {method: httpclient.MethodPatch, want: "PATCH"},
			"DELETEは文字列DELETEを表す": {method: httpclient.MethodDelete, want: "DELETE"},
		}

		for name, tc := range cases {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tc.want, string(tc.method))
			})
		}
	})
}

func TestRequest(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("各フィールドを保持できる", func(t *testing.T) {
			t.Parallel()

			req := &httpclient.Request{
				Downstream:     "payment",
				Method:         httpclient.MethodPost,
				URL:            "https://example.com/charge",
				Header:         httpclient.Header{"Content-Type": {"application/json"}},
				Body:           []byte(`{"amount":100}`),
				IdempotencyKey: "key-1",
				AllowRetry:     true,
			}

			assert.Equal(t, httpclient.Downstream("payment"), req.Downstream)
			assert.Equal(t, httpclient.MethodPost, req.Method)
			assert.Equal(t, "https://example.com/charge", req.URL)
			assert.Equal(t, []string{"application/json"}, req.Header["Content-Type"])
			assert.Equal(t, []byte(`{"amount":100}`), req.Body)
			assert.Equal(t, "key-1", req.IdempotencyKey)
			assert.True(t, req.AllowRetry)
		})
	})
}
