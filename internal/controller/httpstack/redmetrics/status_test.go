package redmetrics

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeStatus(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("0は500に補正される", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, http.StatusInternalServerError, normalizeStatus(0))
		})

		t.Run("0以外はそのまま返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, http.StatusOK, normalizeStatus(http.StatusOK))
		})
	})
}

func TestStatusClass(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		code int
		want string
	}{
		{name: "1xx境界", code: 100, want: "1xx"},
		{name: "2xx境界", code: 200, want: "2xx"},
		{name: "3xx境界", code: 300, want: "3xx"},
		{name: "4xx境界", code: 404, want: "4xx"},
		{name: "5xx境界", code: 500, want: "5xx"},
		{name: "範囲未満はunknown", code: 99, want: statusClassUnknown},
		{name: "範囲超過はunknown", code: 600, want: statusClassUnknown},
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tc.want, statusClass(tc.code))
			})
		}
	})
}

func TestStatusCodeLabel(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("正の値は文字列化される", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "200", statusCodeLabel(http.StatusOK))
		})

		t.Run("0以下はunknownになる", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, statusCodeUnknown, statusCodeLabel(0))
		})
	})
}
