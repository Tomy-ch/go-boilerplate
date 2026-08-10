package decimal

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// FuzzParse は、Parse が任意のバイト列に対して panic せず、
// 受理した値は String() を経ても同じ値へ再解釈されることを検証します。
func FuzzParse(f *testing.F) {
	for _, seed := range []string{
		"0",
		"1",
		"-1",
		"19.99",
		"0.000000001",
		"-0.0",
		"1e10",
		strings.Join([]string{"340282366920938", "463374607431768", "211455"}, ""),
		"",
		"abc",
		".",
		"-",
		"1.2.3",
		"１.０",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, s string) {
		d, err := Parse(s)
		if err != nil {
			return
		}

		// 受理された値は自身の文字列表現から同じ値へ戻る（往復の安定性）。
		// ここが破れると、永続化して読み直した金額が元と一致しなくなる。
		again, err := Parse(d.String())
		require.NoError(t, err)
		require.Equal(t, d.String(), again.String())
	})
}
