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

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("1xx下限境界", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "1xx", statusClass(100))
		})

		t.Run("1xx上限境界", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "1xx", statusClass(199))
		})

		t.Run("2xx下限境界", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "2xx", statusClass(200))
		})

		t.Run("2xx上限境界", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "2xx", statusClass(299))
		})

		t.Run("3xx下限境界", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "3xx", statusClass(300))
		})

		t.Run("3xx上限境界", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "3xx", statusClass(399))
		})

		t.Run("4xx下限境界", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "4xx", statusClass(400))
		})

		t.Run("4xx中間", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "4xx", statusClass(404))
		})

		t.Run("4xx上限境界", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "4xx", statusClass(499))
		})

		t.Run("5xx下限境界", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "5xx", statusClass(500))
		})

		t.Run("5xx上限境界", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "5xx", statusClass(599))
		})

		t.Run("範囲未満はunknown", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, statusClassUnknown, statusClass(99))
		})

		t.Run("範囲超過はunknown", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, statusClassUnknown, statusClass(600))
		})
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
