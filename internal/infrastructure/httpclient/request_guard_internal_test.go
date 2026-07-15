package httpclient

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test_buildRequest は、自前型 Request から net/http のリクエストが正しく組み立つことを検証します。
func Test_buildRequest(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("method・URL・header・body・冪等性キーを反映して net/http.Request を組み立てる", func(t *testing.T) {
			t.Parallel()

			req := &Request{
				downstream:     "sample",
				method:         MethodPost(),
				url:            "http://example.com/charge",
				header:         Header{"Content-Type": {"application/json"}},
				body:           []byte(`{"v":1}`),
				idempotencyKey: "key-123",
			}

			httpReq, err := buildRequest(context.Background(), req)

			require.NoError(t, err)
			require.NotNil(t, httpReq)
			assert.Equal(t, http.MethodPost, httpReq.Method)
			assert.Equal(t, "http://example.com/charge", httpReq.URL.String())
			assert.Equal(t, "application/json", httpReq.Header.Get("Content-Type"))
			assert.Equal(t, "key-123", httpReq.Header.Get("Idempotency-Key"))

			body, err := io.ReadAll(httpReq.Body)
			require.NoError(t, err)
			assert.Equal(t, []byte(`{"v":1}`), body)
		})
	})
}

// Test_client_Do は、型で防げない Request の precondition（validate）に対する Do の防御的ガードを
// 検証します。空 Downstream / ゼロ値 Method / AllowRetry の空 key はいずれも個別 sentinel で弾かれます。
//
// ガードは Do の最初の文で registry / network に触れる前に返るため、ゼロ値 client で足ります。
func Test_client_Do(t *testing.T) {
	t.Parallel()

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Downstreamが空の場合、ErrInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()

			c := &client{}
			req := &Request{
				downstream: "",
				method:     MethodGet(),
				url:        "http://example.com",
			}

			resp, err := c.Do(context.Background(), req)

			require.ErrorIs(t, err, errDownstreamRequired)
			assert.Nil(t, resp)
		})

		t.Run("Methodがゼロ値の場合、ErrInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()

			c := &client{}
			req := &Request{
				downstream: "empty-method",
				method:     Method{},
				url:        "http://example.com",
			}

			resp, err := c.Do(context.Background(), req)

			require.ErrorIs(t, err, errMethodRequired)
			assert.Nil(t, resp)
		})

		t.Run("AllowRetryありでIdempotencyKeyが空の場合、ErrInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()

			c := &client{}
			req := &Request{
				downstream: "retry",
				method:     MethodPost(),
				url:        "http://example.com",
				allowRetry: true,
			}

			resp, err := c.Do(context.Background(), req)

			require.ErrorIs(t, err, errIdempotencyKeyRequired)
			assert.Nil(t, resp)
		})
	})
}
