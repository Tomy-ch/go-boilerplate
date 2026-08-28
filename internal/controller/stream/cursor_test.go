package stream

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	rt "go-boilerplate/internal/usecase/boundary/realtime"
)

func Test_resolveCursor(t *testing.T) {
	t.Parallel()

	lastEventID, after := "12", "7"

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Last-Event-IDがあればafterより優先する", func(t *testing.T) {
			t.Parallel()
			got, err := resolveCursor(&lastEventID, &after, 3)
			require.NoError(t, err)
			assert.Equal(t, rt.Sequence(12), got)
		})

		t.Run("Last-Event-IDが無ければafterを使う", func(t *testing.T) {
			t.Parallel()
			got, err := resolveCursor(nil, &after, 3)
			require.NoError(t, err)
			assert.Equal(t, rt.Sequence(7), got)
		})

		t.Run("どちらも無ければticketの初期位置を使う", func(t *testing.T) {
			t.Parallel()
			got, err := resolveCursor(nil, nil, 3)
			require.NoError(t, err)
			assert.Equal(t, rt.Sequence(3), got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Last-Event-IDが不正ならafterがあってもErrCursorMalformed", func(t *testing.T) {
			t.Parallel()
			bad := "x"
			_, err := resolveCursor(&bad, &after, 3)
			require.ErrorIs(t, err, ErrCursorMalformed)
		})
	})
}

func Test_parseCursor(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("0はstreamの先頭", func(t *testing.T) {
			t.Parallel()
			got, err := parseCursor("0")
			require.NoError(t, err)
			assert.Equal(t, rt.Sequence(0), got)
		})

		t.Run("int64の最大値まで受け付ける", func(t *testing.T) {
			t.Parallel()
			got, err := parseCursor("9223372036854775807")
			require.NoError(t, err)
			assert.Equal(t, rt.Sequence(9223372036854775807), got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("空文字はErrCursorMalformed", func(t *testing.T) {
			t.Parallel()
			_, err := parseCursor("")
			require.ErrorIs(t, err, ErrCursorMalformed)
		})

		t.Run("負数はErrCursorMalformed", func(t *testing.T) {
			t.Parallel()
			_, err := parseCursor("-1")
			require.ErrorIs(t, err, ErrCursorMalformed)
		})

		t.Run("符号付きと先頭ゼロはErrCursorMalformed", func(t *testing.T) {
			t.Parallel()
			_, err := parseCursor("+1")
			require.ErrorIs(t, err, ErrCursorMalformed)
			_, err = parseCursor("007")
			require.ErrorIs(t, err, ErrCursorMalformed)
		})

		t.Run("int64を超える値はErrCursorMalformed", func(t *testing.T) {
			t.Parallel()
			_, err := parseCursor("9223372036854775808")
			require.ErrorIs(t, err, ErrCursorMalformed)
		})
	})
}
