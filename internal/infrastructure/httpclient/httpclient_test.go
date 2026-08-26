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

func TestMethodGet(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("メソッド文字列GETを持つ非ゼロ値のMethodを返す", func(t *testing.T) {
			t.Parallel()

			m := httpclient.MethodGet()

			assert.Equal(t, "GET", m.String())
			assert.NotEqual(t, httpclient.Method{}, m)
		})
	})
}

func TestMethodPost(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("メソッド文字列POSTを持つ非ゼロ値のMethodを返す", func(t *testing.T) {
			t.Parallel()

			m := httpclient.MethodPost()

			assert.Equal(t, "POST", m.String())
			assert.NotEqual(t, httpclient.Method{}, m)
		})
	})
}

func TestMethodPut(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("メソッド文字列PUTを持つ非ゼロ値のMethodを返す", func(t *testing.T) {
			t.Parallel()

			m := httpclient.MethodPut()

			assert.Equal(t, "PUT", m.String())
			assert.NotEqual(t, httpclient.Method{}, m)
		})
	})
}

func TestMethodPatch(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("メソッド文字列PATCHを持つ非ゼロ値のMethodを返す", func(t *testing.T) {
			t.Parallel()

			m := httpclient.MethodPatch()

			assert.Equal(t, "PATCH", m.String())
			assert.NotEqual(t, httpclient.Method{}, m)
		})
	})
}

func TestMethodDelete(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("メソッド文字列DELETEを持つ非ゼロ値のMethodを返す", func(t *testing.T) {
			t.Parallel()

			m := httpclient.MethodDelete()

			assert.Equal(t, "DELETE", m.String())
			assert.NotEqual(t, httpclient.Method{}, m)
		})
	})
}

func TestMethod_String(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("定義済みメソッドはそれぞれ対応するHTTPメソッド文字列を返す", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, "GET", httpclient.MethodGet().String())
			assert.Equal(t, "POST", httpclient.MethodPost().String())
			assert.Equal(t, "PUT", httpclient.MethodPut().String())
			assert.Equal(t, "PATCH", httpclient.MethodPatch().String())
			assert.Equal(t, "DELETE", httpclient.MethodDelete().String())
		})

		t.Run("ゼロ値のMethodは空文字を返す", func(t *testing.T) {
			t.Parallel()

			assert.Empty(t, httpclient.Method{}.String())
		})
	})
}

func TestWithHeader(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("渡したヘッダをRequestに設定する", func(t *testing.T) {
			t.Parallel()

			header := httpclient.Header{"X-Api-Key": {"k1"}, "Accept": {"application/json"}}
			req := httpclient.NewRequest(
				httpclient.MethodGet(), "rates", "https://example.com/rates", httpclient.WithHeader(header))

			assert.Equal(t, header, req.Header())
		})

		t.Run("後勝ちで適用され先に指定したヘッダを置き換える", func(t *testing.T) {
			t.Parallel()

			req := httpclient.NewRequest(
				httpclient.MethodGet(), "rates", "https://example.com/rates",
				httpclient.WithHeader(httpclient.Header{"X-Api-Key": {"old"}}),
				httpclient.WithHeader(httpclient.Header{"X-Api-Key": {"new"}}),
			)

			assert.Equal(t, []string{"new"}, req.Header()["X-Api-Key"])
		})
	})
}

func TestWithBody(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("渡したボディをRequestに設定する", func(t *testing.T) {
			t.Parallel()

			req := httpclient.NewRequest(
				httpclient.MethodPost(), "pay", "https://example.com/charge",
				httpclient.WithBody([]byte(`{"amount":100}`)))

			assert.Equal(t, []byte(`{"amount":100}`), req.Body())
		})

		t.Run("ボディのみを設定し冪等性キーとretry許可には影響しない", func(t *testing.T) {
			t.Parallel()

			req := httpclient.NewRequest(
				httpclient.MethodPost(), "pay", "https://example.com/charge", httpclient.WithBody([]byte("x")))

			assert.Empty(t, req.IdempotencyKey())
			assert.False(t, req.AllowRetry())
		})
	})
}

func TestWithIdempotencyKey(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("渡した冪等性キーを設定しretryは許可しない", func(t *testing.T) {
			t.Parallel()

			req := httpclient.NewRequest(
				httpclient.MethodPost(), "pay", "https://example.com/charge",
				httpclient.WithIdempotencyKey("key-1"))

			assert.Equal(t, "key-1", req.IdempotencyKey())
			assert.False(t, req.AllowRetry())
		})
	})
}

func TestWithRetry(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("retry許可と冪等性キーを同時に設定する", func(t *testing.T) {
			t.Parallel()

			req := httpclient.NewRequest(
				httpclient.MethodPost(), "pay", "https://example.com/charge", httpclient.WithRetry("key-2"))

			assert.True(t, req.AllowRetry())
			assert.Equal(t, "key-2", req.IdempotencyKey())
		})

		t.Run("空キーを渡した場合もretry許可だけが立ちキーは空のままになる", func(t *testing.T) {
			t.Parallel()

			req := httpclient.NewRequest(
				httpclient.MethodPost(), "pay", "https://example.com/charge", httpclient.WithRetry(""))

			assert.True(t, req.AllowRetry())
			assert.Empty(t, req.IdempotencyKey())
		})
	})
}

func TestRequest_Downstream(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("生成時に渡した論理依存名を返す", func(t *testing.T) {
			t.Parallel()

			req := httpclient.NewRequest(httpclient.MethodGet(), "rates", "https://example.com/rates")

			assert.Equal(t, httpclient.Downstream("rates"), req.Downstream())
		})
	})
}

func TestRequest_Method(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("生成時に渡したHTTPメソッドを返す", func(t *testing.T) {
			t.Parallel()

			req := httpclient.NewRequest(httpclient.MethodDelete(), "rates", "https://example.com/rates")

			assert.Equal(t, httpclient.MethodDelete(), req.Method())
		})
	})
}

func TestRequest_URL(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("生成時に渡したURLを返す", func(t *testing.T) {
			t.Parallel()

			req := httpclient.NewRequest(httpclient.MethodGet(), "rates", "https://example.com/rates?base=USD")

			assert.Equal(t, "https://example.com/rates?base=USD", req.URL())
		})
	})
}

func TestRequest_Header(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("設定したヘッダを返す", func(t *testing.T) {
			t.Parallel()

			req := httpclient.NewRequest(
				httpclient.MethodGet(), "rates", "https://example.com/rates",
				httpclient.WithHeader(httpclient.Header{"X-Api-Key": {"k1"}}))

			assert.Equal(t, []string{"k1"}, req.Header()["X-Api-Key"])
		})

		t.Run("ヘッダ未設定の場合はnilを返す", func(t *testing.T) {
			t.Parallel()

			req := httpclient.NewRequest(httpclient.MethodGet(), "rates", "https://example.com/rates")

			assert.Nil(t, req.Header())
		})

		t.Run("返したヘッダをmapごと値スライスごと変更しても内部状態は変わらない", func(t *testing.T) {
			t.Parallel()

			req := httpclient.NewRequest(
				httpclient.MethodGet(), "rates", "https://example.com/rates",
				httpclient.WithHeader(httpclient.Header{"X-Api-Key": {"k1"}}))

			got := req.Header()
			got["X-Api-Key"][0] = "tampered"
			got["X-Injected"] = []string{"x"}

			assert.Equal(t, []string{"k1"}, req.Header()["X-Api-Key"])
			assert.NotContains(t, req.Header(), "X-Injected")
		})
	})
}

func TestRequest_Body(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("設定したボディを返す", func(t *testing.T) {
			t.Parallel()

			req := httpclient.NewRequest(
				httpclient.MethodPost(), "pay", "https://example.com/charge",
				httpclient.WithBody([]byte("original")))

			assert.Equal(t, []byte("original"), req.Body())
		})

		t.Run("ボディ未設定の場合はnilを返す", func(t *testing.T) {
			t.Parallel()

			req := httpclient.NewRequest(httpclient.MethodPost(), "pay", "https://example.com/charge")

			assert.Nil(t, req.Body())
		})

		t.Run("返したボディを変更しても内部状態は変わらない", func(t *testing.T) {
			t.Parallel()

			req := httpclient.NewRequest(
				httpclient.MethodPost(), "pay", "https://example.com/charge",
				httpclient.WithBody([]byte("original")))

			body := req.Body()
			body[0] = 'X'

			assert.Equal(t, []byte("original"), req.Body())
		})
	})
}

func TestRequest_IdempotencyKey(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("設定した冪等性キーを返す", func(t *testing.T) {
			t.Parallel()

			req := httpclient.NewRequest(
				httpclient.MethodPost(), "pay", "https://example.com/charge",
				httpclient.WithIdempotencyKey("key-1"))

			assert.Equal(t, "key-1", req.IdempotencyKey())
		})

		t.Run("冪等性キー未設定の場合は空文字を返す", func(t *testing.T) {
			t.Parallel()

			req := httpclient.NewRequest(httpclient.MethodPost(), "pay", "https://example.com/charge")

			assert.Empty(t, req.IdempotencyKey())
		})
	})
}

func TestRequest_AllowRetry(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("WithRetryで生成した場合はtrueを返す", func(t *testing.T) {
			t.Parallel()

			req := httpclient.NewRequest(
				httpclient.MethodPost(), "pay", "https://example.com/charge", httpclient.WithRetry("key-2"))

			assert.True(t, req.AllowRetry())
		})

		t.Run("retry許可を指定しない場合はfalseを返す", func(t *testing.T) {
			t.Parallel()

			req := httpclient.NewRequest(
				httpclient.MethodPost(), "pay", "https://example.com/charge",
				httpclient.WithIdempotencyKey("key-1"))

			assert.False(t, req.AllowRetry())
		})
	})
}
