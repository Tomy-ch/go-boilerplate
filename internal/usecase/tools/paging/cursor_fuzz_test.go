package paging

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// FuzzNewCursor は、NewCursor が任意の不透明カーソル文字列に対して panic せず、
// 受理した場合はソートキーのタプルを取り出せることを検証します。
//
//	カーソルは URL に載る外部入力であり、base64 と JSON の二段復号を通るため、
//	どちらの層の異常も NewCursor がエラーとして閉じ込める必要があります。
func FuzzNewCursor(f *testing.F) {
	for _, seed := range []string{
		"",
		EncodeCursor("2026-01-01T00:00:00Z"),
		EncodeCursor("2026-01-01T00:00:00Z", "b1d4e0f2-3c5a-4b6d-8e7f-1a2b3c4d5e6f"),
		"not-base64!!",
		"e30",
		"W10",
		"////",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, after string) {
		c, err := NewCursor(&after, nil)
		if err != nil {
			return
		}

		require.NotNil(t, c)
		require.LessOrEqual(t, c.Limit(), maxPerPage)
		require.Positive(t, c.Limit())

		// 受理されたカーソルのキーは再エンコードして再解釈しても同じタプルへ戻る。
		if c.HasCursor() {
			keys := c.Keys()
			round := EncodeCursor(keys...)
			back, err := NewCursor(&round, nil)
			require.NoError(t, err)
			require.Equal(t, keys, back.Keys())
		}
	})
}
