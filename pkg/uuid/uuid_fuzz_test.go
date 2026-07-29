package uuid

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// FuzzParse は、Parse が任意の文字列に対して panic せず、
// 受理した値は String() を経ても同じ値へ再解釈されることを検証します。
func FuzzParse(f *testing.F) {
	for _, seed := range []string{
		"b1d4e0f2-3c5a-4b6d-8e7f-1a2b3c4d5e6f",
		"B1D4E0F2-3C5A-4B6D-8E7F-1A2B3C4D5E6F",
		"00000000-0000-0000-0000-000000000000",
		"urn:uuid:b1d4e0f2-3c5a-4b6d-8e7f-1a2b3c4d5e6f",
		"b1d4e0f23c5a4b6d8e7f1a2b3c4d5e6f",
		"",
		"not-a-uuid",
		"b1d4e0f2-3c5a-4b6d-8e7f-1a2b3c4d5e6",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, s string) {
		u, err := Parse(s)
		if err != nil {
			return
		}

		// 受理された ID は自身の文字列表現から同じ値へ戻る。ここが破れると、
		// URL に載せた ID を読み直したときに別のレコードを指しうる。
		again, err := Parse(u.String())
		require.NoError(t, err)
		require.Equal(t, u.String(), again.String())
	})
}
